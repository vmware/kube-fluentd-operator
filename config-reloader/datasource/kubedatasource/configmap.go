package kubedatasource

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	kfoListersV1beta1 "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/listers/logs.vdp.vmware.com/v1beta1"
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
	ctx        *context.Context
	cfg        *config.Config
	cfglist    listerv1.ConfigMapLister
	cfgready   func() bool
	nslist     listerv1.NamespaceLister
	updateChan chan time.Time
}

func NewConfigMapDS(ctx context.Context, cfg *config.Config, factory informers.SharedInformerFactory, updateChan chan time.Time) (*ConfigMapDS, error) {
	configMapLister := factory.Core().V1().ConfigMaps().Lister()
	namespaceLister := factory.Core().V1().Namespaces().Lister()

	cmDS := &ConfigMapDS{
		ctx:        &ctx,
		cfg:        cfg,
		cfglist:    configMapLister,
		cfgready:   factory.Core().V1().ConfigMaps().Informer().HasSynced,
		nslist:     namespaceLister,
		updateChan: updateChan,
	}

	factory.Core().V1().ConfigMaps().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(new interface{}) {
			cmDS.handleCMChange(ctx, new)
		},
		UpdateFunc: func(old, new interface{}) {
			cmDS.handleCMChange(ctx, new)
		},
		DeleteFunc: func(new interface{}) {
			cmDS.handleCMChange(ctx, new)
		},
	})

	return cmDS, nil
}

// GetFdlist return nil for this mode because it does not use CRDs:
func (c *ConfigMapDS) GetFdlist() kfoListersV1beta1.FluentdConfigLister {
	return nil
}

// IsReady returns a boolean specifying whether the ConfigMapDS is ready
func (c *ConfigMapDS) IsReady() bool {
	return c.cfgready()
}

// GetFluentdConfig returns the fluentd configs for the given ns extracted
// by the configured ConfigMaps
func (c *ConfigMapDS) GetFluentdConfig(ctx context.Context, namespace string) (string, error) {
	// Detect the configmaps in the namespace which we expect to contain
	// fluentd configuration, then read their contents into a buffer
	configmaps, err := c.fetchConfigMaps(ctx, namespace)
	if err != nil {
		return "", err
	}
	return c.readConfig(configmaps), nil
}

func (c *ConfigMapDS) fetchConfigMaps(ctx context.Context, ns string) ([]*core.ConfigMap, error) {
	configmaps := make([]*core.ConfigMap, 0)
	nsmaps := c.cfglist.ConfigMaps(ns)

	if c.cfg.Datasource == "multimap" {
		// Get all configmaps which match a specified label, but only if we have a selector
		mapslist, err := nsmaps.List(c.cfg.ParsedLabelSelector.AsSelector())
		if err != nil {
			return nil, fmt.Errorf("Failed to list configmaps in namespace '%s': %v", ns, err)
		}
		confMapByName := make(map[string]*core.ConfigMap)
		sortedConfMaps := make([]string, 0, len(mapslist))
		for _, cfgm := range mapslist {
			confMapByName[cfgm.Name] = cfgm
			sortedConfMaps = append(sortedConfMaps, cfgm.Name)
		}
		sort.Strings(sortedConfMaps)
		for _, name := range sortedConfMaps {
			configmaps = append(configmaps, confMapByName[name])
		}
	} else {
		// Get a configmap with a specific name
		mapName, err := c.detectConfigMapName(ctx, ns)
		if err != nil {
			switch err.(type) {
			case *namespaceNotConfigured:
				logrus.Debugf("Could not find a named configmap for namespace: %v", err)
			default:
				logrus.Errorln("Unexpected error occurred while getting configmap name")
				return nil, err
			}
		}
		singlemap, err := nsmaps.Get(mapName)
		if err != nil {
			logrus.Tracef("Failed to retrieve configmap '%s' from namespace '%s': %v", mapName, ns, err)
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
func (c *ConfigMapDS) detectConfigMapName(ctx context.Context, ns string) (string, error) {
	namespace, err := c.nslist.Get(ns)
	if err != nil {
		return "", fmt.Errorf("Could not get the details of namespace '%s': %v", ns, err)
	}

	configMapName := namespace.Annotations[c.cfg.AnnotConfigmapName]
	if configMapName == "" {
		if c.cfg.DefaultConfigmapName != "" {
			configMapName = c.cfg.DefaultConfigmapName
			logrus.Tracef("Using default configmap name ('%s') for namespace '%s'", configMapName, ns)
		} else {
			logrus.Tracef("Could not find named configmap in namespace '%s': not annotated with '%s'", ns, c.cfg.AnnotConfigmapName)
			return "", &namespaceNotConfigured{Namespace: ns}
		}
	}

	return configMapName, nil
}

func (c *ConfigMapDS) handleCMChange(ctx context.Context, obj interface{}) {
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
		if len(cmLabels) == 0 || !areLabelsInAllowList(c.cfg.ParsedLabelSelector, labels.Set(cmLabels)) {
			return
		}
	} else {
		mapName, err := c.detectConfigMapName(ctx, object.GetNamespace())
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

// areLabelsInAllowList verifies if the provided label list
// is in the provided allowlist and returns true, otherwise false.
func areLabelsInAllowList(labels, allowlist labels.Set) bool {
	if len(allowlist) == 0 {
		return true
	}

	for k, v := range labels {
		value, ok := allowlist[k]
		if !ok {
			return false
		}
		if value != v {
			return false
		}
	}
	return true
}
