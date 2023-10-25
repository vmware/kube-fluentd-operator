package datasource

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/template"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource"

	kfoListersV1beta1 "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/listers/logs.vdp.vmware.com/v1beta1"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
)

type kubeInformerConnection struct {
	client        kubernetes.Interface
	confHashes    map[string]string
	mountedLabels map[string][]map[string]string
	cfg           *config.Config
	kubeds        kubedatasource.KubeDS
	nslist        listerv1.NamespaceLister
	podlist       listerv1.PodLister
	cmlist        listerv1.ConfigMapLister
	fdlist        kfoListersV1beta1.FluentdConfigLister
	updateChan    chan time.Time
}

// NewKubernetesInformerDatasource builds a new Datasource from the provided config.
// The returned Datasource uses Informers to efficiently track objects in the kubernetes
// API by watching for updates to a known state.
func NewKubernetesInformerDatasource(ctx context.Context, cfg *config.Config, updateChan chan time.Time) (Datasource, error) {
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
	cmLister := factory.Core().V1().ConfigMaps().Lister()

	var kubeds kubedatasource.KubeDS
	fluentdconfigDSLister :=
		&kubedatasource.FluentdConfigDS{
			Fdlist: nil,
		}
	if cfg.Datasource == "crd" {
		kubeds, err = kubedatasource.NewFluentdConfigDS(ctx, cfg, kubeCfg, updateChan)
		if err != nil {
			return nil, err
		}
		fluentdconfigDSLister =
			&kubedatasource.FluentdConfigDS{
				Fdlist: kubeds.GetFdlist(),
			}
	} else {
		if cfg.CRDMigrationMode {
			kubeds, err = kubedatasource.NewMigrationModeDS(ctx, cfg, kubeCfg, factory, updateChan)
			if err != nil {
				return nil, err
			}
			fluentdconfigDSLister =
				&kubedatasource.FluentdConfigDS{
					Fdlist: kubeds.GetFdlist(),
				}
		} else {
			kubeds, err = kubedatasource.NewConfigMapDS(ctx, cfg, factory, updateChan)
			if err != nil {
				return nil, err
			}
		}
	}

	factory.Start(nil)
	if !cache.WaitForCacheSync(nil,
		factory.Core().V1().Namespaces().Informer().HasSynced,
		factory.Core().V1().Pods().Informer().HasSynced,
		factory.Core().V1().ConfigMaps().Informer().HasSynced,
		kubeds.IsReady) {
		return nil, fmt.Errorf("failed to sync local informer with upstream Kubernetes API")
	}
	logrus.Infof("Synced local informer with upstream Kubernetes API")

	kubeInfoCx := &kubeInformerConnection{
		client:        client,
		confHashes:    make(map[string]string),
		mountedLabels: make(map[string][]map[string]string),
		cfg:           cfg,
		kubeds:        kubeds,
		nslist:        namespaceLister,
		podlist:       podLister,
		cmlist:        cmLister,
		fdlist:        fluentdconfigDSLister.Fdlist,
		updateChan:    updateChan,
	}

	factory.Core().V1().Pods().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			kubeInfoCx.handlePodChange(ctx, obj)
		},
		UpdateFunc: func(old, obj interface{}) {
		},
		DeleteFunc: func(obj interface{}) {
			kubeInfoCx.handlePodChange(ctx, obj)
		},
	})

	return kubeInfoCx, nil
}

// GetNamespaces queries the configured Kubernetes API to generate a list of NamespaceConfig objects.
// It uses options from the configuration to determine which namespaces to inspect and which resources
// within those namespaces contain fluentd configuration.
func (d *kubeInformerConnection) GetNamespaces(ctx context.Context) ([]*NamespaceConfig, error) {
	// Get a list of the namespaces which may contain fluentd configuration:
	nses, err := d.discoverNamespaces(ctx)
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

		if d.cfg.AllowLabel != "" {
			template.SetAllowLabel(d.cfg.AllowLabel)
		}
		if d.cfg.AllowLabelAnnotation != "" {
			if label := nsobj.GetAnnotations()[d.cfg.AllowLabelAnnotation]; label != "" {
				template.SetAllowLabel(label)
			}
		}

		configdata, err := d.kubeds.GetFluentdConfig(ctx, ns)
		if err != nil {
			return nil, err
		}
		buf := new(strings.Builder)
		if err := template.Render(buf, configdata, map[string]string{
			"Namespace": ns,
		}); err == nil {
			configdata = buf.String()
		}
		if configdata == "" {
			logrus.Infof("Skipping namespace: %v because is empty", ns)
			continue
		}
		fragment, err := fluentd.ParseString(configdata)
		if err != nil {
			logrus.Errorf("Error parsing config for ns %s: %v", ns, err)
			continue
		}

		var mountedLabels []map[string]string
		for _, frag := range fragment {
			if frag.Name == "source" && frag.Type() == "mounted-file" {
				paramLabels := frag.Param("labels")
				paramLabels = util.TrimTrailingComment(paramLabels)
				currLabels, err := util.ParseTagToLabels(fmt.Sprintf("$labels(%s)", paramLabels))
				if err != nil {
					return nil, err
				}
				mountedLabels = append(mountedLabels, currLabels)
			}
		}

		d.updateMountedLabels(ns, mountedLabels)

		// Create a compact representation of the pods running in the namespace
		// under consideration
		pods, err := d.podlist.Pods(ns).List(labels.NewSelector())
		if err != nil {
			logrus.Errorf("Error listing pod in ns %s: %v", ns, err)
			continue
		}
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
			PreviousConfigHash: d.confHashes[ns],
			Labels:             nsobj.Labels,
			MiniContainers:     minis,
		})
	}

	return nsconfigs, nil
}

// WriteCurrentConfigHash is a setter for the hashtable maintained by this Datasource
func (d *kubeInformerConnection) WriteCurrentConfigHash(namespace string, hash string) {
	d.confHashes[namespace] = hash
}

func (d *kubeInformerConnection) updateMountedLabels(namespace string, labels []map[string]string) {
	d.mountedLabels[namespace] = labels
}

// UpdateStatus updates a namespace's status annotation with the latest result
// from the config generator.
func (d *kubeInformerConnection) UpdateStatus(ctx context.Context, namespace string, status string) {
	ns, err := d.client.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
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

	_, err = d.client.CoreV1().Namespaces().Update(ctx, ns, metav1.UpdateOptions{})

	logrus.Debugf("Saving status annotation to namespace %s: %+v", namespace, err)
	// errors.IsConflict is safe to ignore since multiple log-routers try update at same time
	// (only 1 router can update this unique ResourceVersion, no need to retry, each router is a retry process):
	if err != nil && !errors.IsConflict(err) {
		logrus.Infof("Cannot set error status on namespace %s: %+v", namespace, err)
	}
}

// discoverNamespaces constructs a list of namespaces to inspect for fluentd
// configuration, using the configured list if provided, or find namespaces based on labels if provided in --namespace-selector flag, otherwise find only
// namespaces that have fluentd configmaps based on default name, and if that fails
// find all namespace and iterrate through them.
func (d *kubeInformerConnection) discoverNamespaces(ctx context.Context) ([]string, error) {
	var namespaces []string
	if len(d.cfg.Namespaces) != 0 {
		namespaces = d.cfg.Namespaces
	} else if d.cfg.NamespaceSelector != "" {
		// create label selector to list namespaces based on labels provided in namespace-selector flag
		nsLabelSelector, err := labels.Parse(d.cfg.NamespaceSelector)
		if err != nil {
			return nil, err
		}

		nses, err := d.nslist.List(nsLabelSelector)
		if err != nil {
			return nil, err
		}
		for _, ns := range nses {
			namespaces = append(namespaces, ns.Name)
		}

	} else {
		if d.cfg.Datasource == "crd" {
			logrus.Infof("Discovering only namespaces that have fluentdconfig crd defined.")
			nsList, err := d.discoverFluentdConfigNamespaces()
			if err != nil {
				return nil, err
			}
			namespaces = nsList
		} else {
			// Find the configmaps that exist on this cluster to find namespaces:
			confMapsList, err := d.cmlist.List(labels.NewSelector())
			if err != nil {
				return nil, fmt.Errorf("failed to list all configmaps in cluster: %v", err)
			}
			// If default configmap name is defined get all namespaces for those configmaps:
			if d.cfg.DefaultConfigmapName != "" {
				for _, cfmap := range confMapsList {
					if cfmap.Name == d.cfg.DefaultConfigmapName {
						namespaces = append(namespaces, cfmap.Namespace)
					} else {
						// We need to find configmaps that honor the global annotation for configmaps:
						configMapNamespace, _ := d.nslist.Get(cfmap.Namespace)
						configMapName := configMapNamespace.Annotations[d.cfg.AnnotConfigmapName]
						if configMapName != "" {
							namespaces = append(namespaces, cfmap.Namespace)
						}
					}
				}
				if d.cfg.CRDMigrationMode {
					nsList, err := d.discoverFluentdConfigNamespaces()
					if err != nil {
						return nil, err
					}
					namespaces = append(namespaces, nsList...)
				}
			} else {
				// get all namespaces and iterrate through them like before:
				nses, err := d.nslist.List(labels.NewSelector())
				if err != nil {
					return nil, fmt.Errorf("failed to list all namespaces in cluster: %v", err)
				}
				namespaces = make([]string, 0)
				for _, ns := range nses {
					namespaces = append(namespaces, ns.Name)
				}
			}
		}
	}
	// Remove duplicates (crds can be many in single namespace):
	nsKeys := make(map[string]bool)
	nsList := []string{}
	for _, ns := range namespaces {
		if _, value := nsKeys[ns]; !value {
			nsKeys[ns] = true
			nsList = append(nsList, ns)
		}
	}
	// Sort the namespaces:
	sort.Strings(nsList)
	return nsList, nil
}

func (d *kubeInformerConnection) handlePodChange(ctx context.Context, obj interface{}) {
	mObj := obj.(*core.Pod)
	logrus.Tracef("Detected pod change %s in namespace: %s", mObj.GetName(), mObj.GetNamespace())
	configdata, err := d.kubeds.GetFluentdConfig(ctx, mObj.GetNamespace())
	buf := new(strings.Builder)
	if err := template.Render(buf, configdata, map[string]string{
		"Namespace": mObj.GetNamespace(),
	}); err == nil {
		configdata = buf.String()
	}
	nsConfigStr := fmt.Sprintf("%#v", configdata)

	if err == nil {
		if strings.Contains(nsConfigStr, "mounted-file") {
			podLabels := mObj.GetLabels()
			mountedLabel := d.mountedLabels[mObj.GetNamespace()]
			for _, container := range mObj.Spec.Containers {
				if matchAny(podLabels, mountedLabel, container.Name) {
					logrus.Infof("Detected mounted-file pod change %s in namespace: %s", mObj.GetName(), mObj.GetNamespace())
					select {
					case d.updateChan <- time.Now():
					default:
					}
				}
			}
		}
	}
}

func matchAny(contLabels map[string]string, mountedLabelsInNs []map[string]string, name string) bool {
	for _, mountedLabels := range mountedLabelsInNs {
		if util.Match(mountedLabels, contLabels, name) {
			return true
		}
	}

	return false
}

func (d *kubeInformerConnection) discoverFluentdConfigNamespaces() ([]string, error) {
	if d.fdlist == nil {
		return nil, fmt.Errorf("failed to initialize the fluentdconfig crd client, d.fclient = nil")
	}
	fcList, err := d.fdlist.List(labels.NewSelector())
	if err != nil {
		return nil, fmt.Errorf("failed to list all fluentdconfig crds in cluster: %v", err)
	}
	nsList := make([]string, 0)
	for _, crd := range fcList {
		nsList = append(nsList, crd.Namespace)
	}
	logrus.Debugf("Returned these namespaces for fluentdconfig crds: %v", nsList)
	return nsList, nil
}
