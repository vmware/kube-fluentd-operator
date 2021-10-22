package kubedatasource

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	kfo "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/apis/logs.vdp.vmware.com/v1beta1"
	kfoClient "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/clientset/versioned"
	kfoInformers "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/informers/externalversions"
	kfoListersV1beta1 "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/listers/logs.vdp.vmware.com/v1beta1"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/crd"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type FluentdConfigDS struct {
	Cfg        *config.Config
	Fdlist     kfoListersV1beta1.FluentdConfigLister
	Fdready    func() bool
	UpdateChan chan time.Time
}

func NewFluentdConfigDS(ctx context.Context, cfg *config.Config, kubeCfg *rest.Config, updateChan chan time.Time) (*FluentdConfigDS, error) {
	kfocli, err := kfoClient.NewForConfig(kubeCfg)
	if err != nil {
		return nil, err
	}

	factory := kfoInformers.NewSharedInformerFactory(kfocli, 0)
	fluentdConfigLister := factory.Logs().V1beta1().FluentdConfigs().Lister()

	fdDS := &FluentdConfigDS{
		Cfg:        cfg,
		Fdlist:     fluentdConfigLister,
		Fdready:    factory.Logs().V1beta1().FluentdConfigs().Informer().HasSynced,
		UpdateChan: updateChan,
	}

	factory.Logs().V1beta1().FluentdConfigs().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: fdDS.handleFDChange,
		UpdateFunc: func(old, new interface{}) {
			fdDS.handleFDChange(new)
		},
		DeleteFunc: fdDS.handleFDChange,
	})

	// Verify CRD availability
	if err := crd.CheckAndInstallCRD(ctx, kubeCfg); err != nil {
		return nil, err
	}

	factory.Start(nil)

	return fdDS, nil
}

// GetFdList returns the lister for fluentconfigs informer:
func (f *FluentdConfigDS) GetFdlist() kfoListersV1beta1.FluentdConfigLister {
	return f.Fdlist
}

// IsReady returns a boolean specifying whether the FluentdConfigDS is ready
func (f *FluentdConfigDS) IsReady() bool {
	return f.Fdready()
}

// GetFluentdConfig returns the fluentd configs for the given ns extracted
// by the configured FluentdConfigs k8s resources
func (f *FluentdConfigDS) GetFluentdConfig(ctx context.Context, namespace string) (string, error) {
	// Grab all FluentdConfigs k8s resources in the given ns
	fluentdConfigs, err := f.Fdlist.FluentdConfigs(namespace).List(labels.Everything())
	if err != nil {
		return "", err
	}

	fcByName := make(map[string]*kfo.FluentdConfig)
	sortedFluentdConfigs := make([]string, 0, len(fluentdConfigs))
	for _, fc := range fluentdConfigs {
		fcByName[fc.Name] = fc
		sortedFluentdConfigs = append(sortedFluentdConfigs, fc.Name)
	}
	sort.Strings(sortedFluentdConfigs)

	// Extract fluentd
	configData := make([]string, 0, len(sortedFluentdConfigs))
	for _, name := range sortedFluentdConfigs {
		fd := fcByName[name]
		logrus.Debugf("loaded config data from fluentdconfig: %s/%s", fd.ObjectMeta.Namespace, fd.ObjectMeta.Name)
		configData = append(configData, fd.Spec.FluentConf)
	}

	// Concatenate all namespace's configs
	return strings.Join(configData, "\n"), nil
}

// handleFDChange reacts to changes in the FluentdConfigs k8s resources and notifies the
// main controller to re-run the main loop and sync the state
func (f *FluentdConfigDS) handleFDChange(obj interface{}) {
	// If the controller is monitoring all namespaces, it needs to react
	// to all notifications of changes to FluentdConfigs resources.
	// If instead only a subset of namespaces is being monitored, there
	// is no need to run the control loop unless the changed FluentdConfig
	// resource is in one of the monitored namespaces
	if len(f.Cfg.Namespaces) != 0 {
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

		toProcess := false
		for _, ns := range f.Cfg.Namespaces {
			if object.GetNamespace() == ns {
				toProcess = true
				break
			}
		}
		if !toProcess {
			return
		}
	}

	select {
	case f.UpdateChan <- time.Now():
	default:
		// There is already one pending notification. Useless to send another one since, when
		// the pending one will be processed all new changes will be reloaded.
	}
}
