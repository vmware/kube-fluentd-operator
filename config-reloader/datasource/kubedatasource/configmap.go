package kubedatasource

import (
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"

	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/informers"
	listerv1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/client-go/tools/cache"
)

const (
	entryName = "fluent.conf"
)

type namespaceNotConfigured struct {
	Namespace string
}

func (e *namespaceNotConfigured) Error() string {
	return fmt.Sprintf("Namespace '%s' is not configured", e.Namespace)
}

type ConfigMapDS struct {
	cfg        *config.Config
	cfglist    listerv1.ConfigMapLister
	cfgready   func() bool
	nslist     listerv1.NamespaceLister
	updateChan chan time.Time
}

func NewConfigMapDS(cfg *config.Config, factory informers.SharedInformerFactory, updateChan chan time.Time) (*ConfigMapDS, error) {
	configMapLister := factory.Core().V1().ConfigMaps().Lister()
	namespaceLister := factory.Core().V1().Namespaces().Lister()

	cmDS := &ConfigMapDS{
		cfg:        cfg,
		cfglist:    configMapLister,
		cfgready:   factory.Core().V1().ConfigMaps().Informer().HasSynced,
		nslist:     namespaceLister,
		updateChan: updateChan,
	}

	factory.Core().V1().ConfigMaps().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: cmDS.handleCMChange,
		UpdateFunc: func(old, new interface{}) {
			cmDS.handleCMChange(new)
		},
		DeleteFunc: cmDS.handleCMChange,
	})

	return cmDS, nil
}

// IsReady returns a boolean specifying whether the ConfigMapDS is ready
func (c *ConfigMapDS) IsReady() bool {
	return c.cfgready()
}

// GetFluentdConfig returns the fluentd configs for the given ns extracted
// by the configured ConfigMaps
func (c *ConfigMapDS) GetFluentdConfig(namespace string) (string, error) {
	// Detect the configmaps in the namespace which we expect to contain
	// fluentd configuration, then read their contents into a buffer
	configmaps, err := c.fetchConfigMaps(namespace)
	if err != nil {
		return "", err
	}
	return c.readConfig(configmaps), nil
}

func (c *ConfigMapDS) fetchConfigMaps(ns string) ([]*core.ConfigMap, error) {
	configmaps := make([]*core.ConfigMap, 0)
	nsmaps := c.cfglist.ConfigMaps(ns)

	if c.cfg.Datasource == "multimap" {
		// Get all configmaps which match a specified label, but only if we have a selector
		mapslist, err := nsmaps.List(c.cfg.ParsedLabelSelector.AsSelector())
		if err != nil {
			return nil, fmt.Errorf("Failed to list configmaps in namespace '%s': %v", ns, err)
		}
		configmaps = append(configmaps, mapslist...)
	} else {
		// Get a configmap with a specific name
		mapName, err := c.detectConfigMapName(ns)
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
func (c *ConfigMapDS) readConfig(configmaps []*core.ConfigMap) string {
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

// detectConfigMapName calculates the expected name of a configmap containing fluentd
// configuration in the provided namespace. If the namespace has been annotated to indicate
// which configmap should be used, then the annotation is respected. If there is no
// annotation, the configuration is consulted for a default name. If no name can
// be found, a custom error type is returned indicating that the namespace should
// be excluded from further processing.
func (c *ConfigMapDS) detectConfigMapName(ns string) (string, error) {
	namespace, err := c.nslist.Get(ns)
	if err != nil {
		return "", fmt.Errorf("Could not get the details of namespace '%s': %v", ns, err)
	}

	configMapName := namespace.Annotations[c.cfg.AnnotConfigmapName]
	if configMapName == "" {
		if c.cfg.DefaultConfigmapName != "" {
			configMapName = c.cfg.DefaultConfigmapName
			logrus.Debugf("Using default configmap name ('%s') for namespace '%s'", configMapName, ns)
		} else {
			logrus.Debugf("Could not find named configmap in namespace '%s': not annotated with '%s'", ns, c.cfg.AnnotConfigmapName)
			return "", &namespaceNotConfigured{Namespace: ns}
		}
	}

	return configMapName, nil
}

func (c *ConfigMapDS) handleCMChange(obj interface{}) {
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

	if len(c.cfg.Namespaces) != 0 {
		toProcess := false
		for _, ns := range c.cfg.Namespaces {
			if object.GetNamespace() == ns {
				toProcess = true
				break
			}
		}
		if !toProcess {
			return
		}
	}

	if c.cfg.Datasource == "multimap" {
		cmLabels := object.GetLabels()
		if len(cmLabels) == 0 || !labels.AreLabelsInWhiteList(c.cfg.ParsedLabelSelector, labels.Set(cmLabels)) {
			return
		}
	} else {
		mapName, err := c.detectConfigMapName(object.GetNamespace())
		if err != nil || object.GetName() != mapName {
			return
		}
	}

	select {
	case c.updateChan <- time.Now():
	default:
		// There is already one pending notification. Useless to send another one since, when
		// the pending one will be processed all new changes will be reloaded.
	}
}
