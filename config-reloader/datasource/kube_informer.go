package datasource

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type namespaceNotConfigured struct {
	Namespace string
}

func (e *namespaceNotConfigured) Error() string {
	return fmt.Sprintf("Namespace '%s' is not configured", e.Namespace)
}

type kubeInformerConnection struct {
	client     kubernetes.Interface
	hashes     map[string]string
	cfg        *config.Config
	cfglist    listerv1.ConfigMapLister
	nslist     listerv1.NamespaceLister
	podlist    listerv1.PodLister
	updateChan chan time.Time
}

// GetNamespaces queries the configured Kubernetes API to generate a list of NamespaceConfig objects.
// It uses options from the configuration to determine which namespaces to inspect and which configmaps
// within those namespaces contain fluentd configuration.
func (d *kubeInformerConnection) GetNamespaces() ([]*NamespaceConfig, error) {

	// Get a list of the namespaces which may contain fluentd configuration
	nses, err := d.discoverNamespaces()
	if err != nil {
		return nil, err
	}

	nsconfigs := make([]*NamespaceConfig, 0)
	for _, ns := range nses {
		// Get the Namespace object associated with a particular name
		nsobj, err := d.nslist.Get(ns)
		if err != nil {
			return nil, err
		}

		// Detect the configmaps in each namespace which we expect to contain
		// fluentd configuration, then read their contents into a buffer
		configmaps, err := d.fetchConfigMaps(ns)
		if err != nil {
			return nil, err
		}
		configdata := d.readConfig(configmaps)

		// Create a compact representation of the pods running in the namespace
		// under consideration
		pods, err := d.podlist.Pods(ns).List(labels.NewSelector())
		podsCopy := make([]core.Pod, len(pods))
		for i, pod := range pods {
			podsCopy[i] = *pod.DeepCopy()
		}
		podList := &core.PodList{
			Items: podsCopy,
		}
		minis := convertPodToMinis(podList)

		// Create a new NamespaceConfig from the data we've processed up to now
		nsconfigs = append(nsconfigs, &NamespaceConfig{
			Name:               ns,
			FluentdConfig:      configdata,
			PreviousConfigHash: d.hashes[ns],
			IsKnownFromBefore:  true,
			Labels:             nsobj.Labels,
			MiniContainers:     minis,
		})
	}

	return nsconfigs, nil
}

// WriteCurrentConfigHash is a setter for the hashtable maintained by this Datasource
func (d *kubeInformerConnection) WriteCurrentConfigHash(namespace string, hash string) {
	d.hashes[namespace] = hash
}

// UpdateStatus patches a namespace to update the status annotation with the latest result
// from the config generator.
func (d *kubeInformerConnection) UpdateStatus(namespace string, status string) {
	patch := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Annotations: map[string]string{
				d.cfg.AnnotStatus: status,
			},
		},
	}

	body, _ := json.Marshal(&patch)
	_, err := d.client.CoreV1().Namespaces().Patch(namespace, types.MergePatchType, body)

	logrus.Debugf("Saving status: %+v, %+v", patch, err)
	if err != nil {
		logrus.Infof("Cannot set error status of %s: %v", namespace, err)
	}
}

// detectConfigMapName calculates the expected name of a configmap containing fluentd
// configuration in the provided namespace. If the namespace has been annotated to indicate
// which configmap should be used, then the annotation is respected. If there is no
// annotation, the configuration is consulted for a default name. If no name can
// be found, a custom error type is returned indicating that the namespace should
// be excluded from further processing.
func (d *kubeInformerConnection) detectConfigMapName(ns string) (string, error) {
	namespace, err := d.nslist.Get(ns)
	if err != nil {
		return "", fmt.Errorf("Could not get the details of namespace '%s': %v", ns, err)
	}

	configMapName := namespace.Annotations[d.cfg.AnnotConfigmapName]
	if configMapName == "" {
		if d.cfg.DefaultConfigmapName != "" {
			configMapName = d.cfg.DefaultConfigmapName
			logrus.Debugf("Using default configmap name ('%s') for namespace '%s'", configMapName, ns)
		} else {
			logrus.Debugf("Could not find named configmap in namespace '%s': not annotated with '%s'", ns, d.cfg.AnnotConfigmapName)
			return "", &namespaceNotConfigured{Namespace: ns}
		}
	}

	return configMapName, nil
}

// discoverNamespaces constructs a list of namespaces to inspect for fluentd
// configuration, using the configured list if provided, otherwise all namespaces are inspected
func (d *kubeInformerConnection) discoverNamespaces() ([]string, error) {
	var namespaces []string
	if len(d.cfg.Namespaces) != 0 {
		namespaces = d.cfg.Namespaces
	} else {
		nses, err := d.nslist.List(labels.NewSelector())
		if err != nil {
			return nil, fmt.Errorf("Failed to list all namespaces: %v", err)
		}
		namespaces = make([]string, 0)
		for _, ns := range nses {
			namespaces = append(namespaces, ns.ObjectMeta.Name)
		}
	}
	return namespaces, nil
}

func (d *kubeInformerConnection) fetchConfigMaps(ns string) ([]*core.ConfigMap, error) {
	configmaps := make([]*core.ConfigMap, 0)
	nsmaps := d.cfglist.ConfigMaps(ns)

	if d.cfg.Datasource == "multimap" {
		// Get all configmaps which match a specified label, but only if we have a selector
		mapslist, err := nsmaps.List(d.cfg.ParsedLabelSelector.AsSelector())
		if err != nil {
			return nil, fmt.Errorf("Failed to list configmaps in namespace '%s': %v", ns, err)
		}
		configmaps = append(configmaps, mapslist...)
	} else {
		// Get a configmap with a specific name
		mapName, err := d.detectConfigMapName(ns)
		if err != nil {
			switch err.(type) {
			case *namespaceNotConfigured:
				logrus.Debugf("Could not find a named configmap for namespace: %v", err)
				break
			default:
				logrus.Errorln("Unexpected error occured while getting configmap name")
				return nil, err
			}
		}
		singlemap, err := nsmaps.Get(mapName)
		if err != nil {
			logrus.Debugf("Failed to retrieve configmap '%s' from namespace '%s': %v", mapName, ns, err)
		}
		if singlemap != nil {
			configmaps = append(configmaps, singlemap)
		}
	}

	return configmaps, nil
}

// readConfig accepts a list of configmaps in a particular namespace, and concatenates their data together
func (d *kubeInformerConnection) readConfig(configmaps []*core.ConfigMap) string {
	configdata := make([]string, 0)
	for _, cm := range configmaps {
		mapData, exists := cm.Data[entryName]
		if exists {
			configdata = append(configdata, mapData)
			logrus.Debugf("Loaded config data from config map: %s/%s", cm.ObjectMeta.Namespace, cm.ObjectMeta.Name)
		} else {
			logrus.Warnf("cannot find entry %s in configmap %s/%s", entryName, cm.ObjectMeta.Namespace, cm.ObjectMeta.Name)
		}
	}
	return strings.Join(configdata, "\n")
}

func (d *kubeInformerConnection) handleCMChange(obj interface{}) {
	var object metav1.Object
	var ok bool
	if object, ok = obj.(metav1.Object); !ok {
		tombstone, ok := obj.(cache.DeletedFinalStateUnknown)
		if !ok {
			logrus.Warnf("error decoding object, invalid type")
			return
		}
		object, ok = tombstone.Obj.(metav1.Object)
		if !ok {
			logrus.Warnf("error decoding object tombstone, invalid type")
			return
		}
	}

	if len(d.cfg.Namespaces) != 0 {
		toProcess := false
		for _, ns := range d.cfg.Namespaces {
			if object.GetNamespace() == ns {
				toProcess = true
				break
			}
		}
		if !toProcess {
			return
		}
	}

	if d.cfg.Datasource == "multimap" {
		cmLabels := object.GetLabels()
		if len(cmLabels) == 0 || !labels.AreLabelsInWhiteList(d.cfg.ParsedLabelSelector, labels.Set(cmLabels)) {
			return
		}
	} else {
		mapName, err := d.detectConfigMapName(object.GetNamespace())
		if err != nil || object.GetName() != mapName {
			return
		}
	}

	select {
	case d.updateChan <- time.Now():
	default:
		// There is already one pending notification. Useless to send another one since, when
		// the pending one will be processed all new changes will be reloaded.
	}
}

// NewKubernetesInformerDatasource builds a new Datasource from the provided config.
// The returned Datasource uses Informers to efficiently track objects in the kubernetes
// API by watching for updates to a known state.
func NewKubernetesInformerDatasource(cfg *config.Config, updateChan chan time.Time) (Datasource, error) {
	kubeConfig := cfg.KubeConfig
	if cfg.KubeConfig == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			kubeConfig = clientcmd.RecommendedHomeFile
		}
	}

	kubeCfg, err := clientcmd.BuildConfigFromFlags(cfg.Master, kubeConfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Connected to cluster at %s", kubeCfg.Host)

	factory := informers.NewSharedInformerFactory(client, 0)
	configMapLister := factory.Core().V1().ConfigMaps().Lister()
	namespaceLister := factory.Core().V1().Namespaces().Lister()
	podLister := factory.Core().V1().Pods().Lister()

	datasource := &kubeInformerConnection{
		client:     client,
		hashes:     make(map[string]string),
		cfg:        cfg,
		cfglist:    configMapLister,
		nslist:     namespaceLister,
		podlist:    podLister,
		updateChan: updateChan,
	}

	factory.Core().V1().ConfigMaps().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: datasource.handleCMChange,
		UpdateFunc: func(old, new interface{}) {
			datasource.handleCMChange(new)
		},
		DeleteFunc: datasource.handleCMChange,
	})

	factory.Start(nil)
	if cache.WaitForCacheSync(nil,
		factory.Core().V1().ConfigMaps().Informer().HasSynced,
		factory.Core().V1().Namespaces().Informer().HasSynced,
		factory.Core().V1().Pods().Informer().HasSynced) == false {
		return nil, fmt.Errorf("Failed to sync local informer with upstream Kubernetes API")
	}
	logrus.Infof("Synced local informer with upstream Kubernetes API")

	return datasource, nil
}
