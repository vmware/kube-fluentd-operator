package datasource

import (
	"fmt"
	"os"
	"sort"
	"time"

	"k8s.io/apimachinery/pkg/labels"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type kubeInformerConnection struct {
	client  kubernetes.Interface
	hashes  map[string]string
	cfg     *config.Config
	kubeds  kubedatasource.KubeDS
	nslist  listerv1.NamespaceLister
	podlist listerv1.PodLister
}

// GetNamespaces queries the configured Kubernetes API to generate a list of NamespaceConfig objects.
// It uses options from the configuration to determine which namespaces to inspect and which resources
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

		configdata, err := d.kubeds.GetFluentdConfig(ns)
		if err != nil {
			return nil, err
		}

		// Create a compact representation of the pods running in the namespace
		// under consideration
		pods, err := d.podlist.Pods(ns).List(labels.NewSelector())
		if err != nil {
			return nil, err
		}
		podsCopy := make([]core.Pod, len(pods))
		for i, pod := range pods {
			podsCopy[i] = *pod.DeepCopy()
		}
		podList := &core.PodList{
			Items: podsCopy,
		}
		minis := convertPodToMinis(podList)

		// If the AnnotStatus is not present on ns then set IsKnownFromBefore to false
		annotStatusExists := false
		if _, ok := nsobj.Annotations[d.cfg.AnnotStatus]; ok {
			annotStatusExists = true
		}

		// Create a new NamespaceConfig from the data we've processed up to now
		nsconfigs = append(nsconfigs, &NamespaceConfig{
			Name:               ns,
			FluentdConfig:      configdata,
			PreviousConfigHash: d.hashes[ns],
			IsKnownFromBefore:  annotStatusExists,
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

// UpdateStatus updates a namespace's status annotation with the latest result
// from the config generator.
func (d *kubeInformerConnection) UpdateStatus(namespace string, status string) {
	ns, err := d.client.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		logrus.Infof("Cannot find namespace to update status for: %v", namespace)
	}

	// update annotations
	annotations := ns.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	statusAnnotationExists := false
	if _, ok := annotations[d.cfg.AnnotStatus]; ok {
		statusAnnotationExists = true
	}

	// check the annotation status key and add if status not blank
	if !statusAnnotationExists && status != "" {
		// not found add it.
		// only add status if the status key is not ""
		annotations[d.cfg.AnnotStatus] = status
	}

	// check if annotation status key exists and remove if status blank
	if statusAnnotationExists && status == "" {
		delete(annotations, d.cfg.AnnotStatus)
	}

	ns.SetAnnotations(annotations)

	_, err = d.client.CoreV1().Namespaces().Update(ns)

	logrus.Debugf("Saving status annotation to namespace %s: %+v", namespace, err)
	if err != nil {
		logrus.Infof("Cannot set error status on namespace %s: %+v", namespace, err)
	}
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
	sort.Strings(namespaces)
	return namespaces, nil
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
	namespaceLister := factory.Core().V1().Namespaces().Lister()
	podLister := factory.Core().V1().Pods().Lister()

	var kubeds kubedatasource.KubeDS
	if cfg.Datasource == "crd" {
		kubeds, err = kubedatasource.NewFluentdConfigDS(cfg, kubeCfg, updateChan)
		if err != nil {
			return nil, err
		}
	} else {
		if cfg.CRDMigrationMode {
			kubeds, err = kubedatasource.NewMigrationModeDS(cfg, kubeCfg, factory, updateChan)
			if err != nil {
				return nil, err
			}
		} else {
			kubeds, err = kubedatasource.NewConfigMapDS(cfg, factory, updateChan)
			if err != nil {
				return nil, err
			}
		}
	}

	factory.Start(nil)
	if !cache.WaitForCacheSync(nil,
		factory.Core().V1().Namespaces().Informer().HasSynced,
		factory.Core().V1().Pods().Informer().HasSynced,
		kubeds.IsReady) {
		return nil, fmt.Errorf("Failed to sync local informer with upstream Kubernetes API")
	}
	logrus.Infof("Synced local informer with upstream Kubernetes API")

	return &kubeInformerConnection{
		client:  client,
		hashes:  make(map[string]string),
		cfg:     cfg,
		kubeds:  kubeds,
		nslist:  namespaceLister,
		podlist: podLister,
	}, nil
}
