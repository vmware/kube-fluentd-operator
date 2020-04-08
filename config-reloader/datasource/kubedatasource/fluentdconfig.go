package kubedatasource

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	kfoClient "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/clientset/versioned"
	kfoInformers "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/informers/externalversions"
	kfoListersV1beta1 "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/listers/logging.csp.vmware.com/v1beta1"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
)

type FluentdConfigDS struct {
	cfg        *config.Config
	fdlist     kfoListersV1beta1.FluentdConfigLister
	fdready    func() bool
	updateChan chan time.Time
}

func NewFluentdConfigDS(cfg *config.Config, kubeCfg *rest.Config, updateChan chan time.Time) (*FluentdConfigDS, error) {
	kfocli, err := kfoClient.NewForConfig(kubeCfg)
	if err != nil {
		return nil, err
	}

	factory := kfoInformers.NewSharedInformerFactory(kfocli, 0)
	fluentdConfigLister := factory.Logging().V1beta1().FluentdConfigs().Lister()

	fdDS := &FluentdConfigDS{
		cfg:        cfg,
		fdlist:     fluentdConfigLister,
		fdready:    factory.Logging().V1beta1().FluentdConfigs().Informer().HasSynced,
		updateChan: updateChan,
	}

	factory.Logging().V1beta1().FluentdConfigs().Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: fdDS.handleFDChange,
		UpdateFunc: func(old, new interface{}) {
			fdDS.handleFDChange(new)
		},
		DeleteFunc: fdDS.handleFDChange,
	})

	// Verify CRD availability
	if err := fdDS.checkAndInstallCRDs(kubeCfg); err != nil {
		return nil, err
	}

	factory.Start(nil)

	return fdDS, nil
}

// IsReady returns a boolean specifying whether the FluentdConfigDS is ready
func (f *FluentdConfigDS) IsReady() bool {
	return f.fdready()
}

// GetFluentdConfig returns the fluentd configs for the given ns extracted
// by the configured FluentdConfigs k8s resources
func (f *FluentdConfigDS) GetFluentdConfig(namespace string) (string, error) {
	// Grab all FluentdConfigs k8s resources in the given ns
	fluentdConfigs, err := f.fdlist.FluentdConfigs(namespace).List(labels.Everything())
	if err != nil {
		return "", err
	}

	// Extract fluentd
	configData := make([]string, 0, len(fluentdConfigs))
	for _, fd := range fluentdConfigs {
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
	if len(f.cfg.Namespaces) != 0 {
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
		for _, ns := range f.cfg.Namespaces {
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
	case f.updateChan <- time.Now():
	default:
		// There is already one pending notification. Useless to send another one since, when
		// the pending one will be processed all new changes will be reloaded.
	}
}

///////////////////////////////////////////
// CRD Management for autoinstall
///////////////////////////////////////////

var fluentdConfigCRD = v1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "fluentdconfigs.logging.csp.vmware.com",
	},
	Spec: v1.CustomResourceDefinitionSpec{
		Group: "logging.csp.vmware.com",
		Names: v1.CustomResourceDefinitionNames{
			Plural: "fluentdconfigs",
			Kind:   "FluentdConfig",
		},
		Scope: v1.NamespaceScoped,
		Versions: []v1.CustomResourceDefinitionVersion{
			v1.CustomResourceDefinitionVersion{
				Name:    "v1beta1",
				Served:  true,
				Storage: true,
				Schema: &v1.CustomResourceValidation{
					OpenAPIV3Schema: &v1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]v1.JSONSchemaProps{
							"spec": v1.JSONSchemaProps{
								Type: "object",
								Properties: map[string]v1.JSONSchemaProps{
									"fluentconf": v1.JSONSchemaProps{
										Type: "string",
									},
								},
							},
						},
					},
				},
			},
		},
	},
}

// checkAndInstallCRDs checks whether the CRDs are already defined in the cluster
// and, if not, installs them and waits for them to be available
func (f *FluentdConfigDS) checkAndInstallCRDs(config *rest.Config) error {
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	if _, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Create(&fluentdConfigCRD); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	logrus.Infof("%s CRD is installed. Checking availability...", fluentdConfigCRD.ObjectMeta.Name)
	if err := f.monitorCRDAvailability(clientset, fluentdConfigCRD.ObjectMeta.Name); err != nil {
		return err
	}
	logrus.Infof("%s CRD is available", fluentdConfigCRD.ObjectMeta.Name)

	return nil
}

func (f *FluentdConfigDS) monitorCRDAvailability(clientset *clientset.Clientset, name string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	for {
		crd, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Get(name, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if f.isCRDStatusOK(crd) {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s CRD has not become available before timeout", name)
		case <-time.After(time.Second):
		}
	}
}

func (f *FluentdConfigDS) isCRDStatusOK(crd *v1.CustomResourceDefinition) bool {
	for _, cond := range crd.Status.Conditions {
		if cond.Type == v1.Established && cond.Status == v1.ConditionTrue {
			return true
		}
	}
	return false
}
