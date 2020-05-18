package crd

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	v1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

type manager interface {
	ApplyCRD() error
	CheckCRD() (bool, error)
	GetCRDName() string
}

// CheckAndInstallCRDs checks whether the CRD is already defined in the cluster
// and, if not, install it and waits for it to be available
// It will automatically install either the legacy v1beta1 CRD or the neq v1 CRD
// based on the available APIs in the Kubernetes cluster
func CheckAndInstallCRD(config *rest.Config) error {
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	var crdManager manager
	v1Available, err := isV1CRDAvailable(config)
	if err != nil {
		return err
	}

	if v1Available {
		crdManager = &v1Manager{clientset}
	} else {
		crdManager = &v1beta1Manager{clientset}
	}

	if err := crdManager.ApplyCRD(); err != nil {
		return err
	}

	logrus.Infof("%s CRD is installed. Checking availability...", crdManager.GetCRDName())
	if err := monitorCRDAvailability(crdManager); err != nil {
		return err
	}
	logrus.Infof("%s CRD is available", crdManager.GetCRDName())

	return nil
}

func isV1CRDAvailable(config *rest.Config) (bool, error) {
	client, err := discovery.NewDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}

	apiGroups, err := client.ServerGroups()
	if err != nil {
		return false, err
	}

	v1Available := false
	v1beta1Available := false
	for _, group := range apiGroups.Groups {
		if group.Name == "apiextensions.k8s.io" {
			for _, version := range group.Versions {
				if version.Version == "v1" {
					v1Available = true
				}
				if version.Version == "v1beta1" {
					v1beta1Available = true
				}
			}
		}
	}

	if !v1Available && !v1beta1Available {
		return false, fmt.Errorf("neither apiextensions.k8s.io/v1beta1 nor apiextensions.k8s.io/v1 are available")
	}

	return v1Available, nil
}

func monitorCRDAvailability(crdManager manager) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	for {
		ok, err := crdManager.CheckCRD()
		if err != nil {
			return err
		}
		if ok {
			return nil
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s CRD has not become available before timeout", crdManager.GetCRDName())
		case <-time.After(time.Second):
		}
	}
}

//////////////////////////////////////////////////
///////////////// v1 CRD Manager /////////////////
//////////////////////////////////////////////////

var fluentdConfigCRD = v1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "fluentdconfigs.logs.vdp.vmware.com",
	},
	Spec: v1.CustomResourceDefinitionSpec{
		Group: "logs.vdp.vmware.com",
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

type v1Manager struct {
	clientset *clientset.Clientset
}

func (m *v1Manager) ApplyCRD() error {
	if _, err := m.clientset.ApiextensionsV1().CustomResourceDefinitions().Create(&fluentdConfigCRD); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (m *v1Manager) CheckCRD() (bool, error) {
	crd, err := m.clientset.ApiextensionsV1().CustomResourceDefinitions().Get(m.GetCRDName(), metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	for _, cond := range crd.Status.Conditions {
		if cond.Type == v1.Established && cond.Status == v1.ConditionTrue {
			return true, nil
		}
	}
	return false, nil
}

func (m *v1Manager) GetCRDName() string {
	return fluentdConfigCRD.ObjectMeta.Name
}

///////////////////////////////////////////////////
/////////////// v1beta1 CRD Manager ///////////////
///////////////////////////////////////////////////

var legacyFluentdConfigCRD = v1beta1.CustomResourceDefinition{
	ObjectMeta: metav1.ObjectMeta{
		Name: "fluentdconfigs.logs.vdp.vmware.com",
	},
	Spec: v1beta1.CustomResourceDefinitionSpec{
		Group: "logs.vdp.vmware.com",
		Names: v1beta1.CustomResourceDefinitionNames{
			Plural: "fluentdconfigs",
			Kind:   "FluentdConfig",
		},
		Validation: &v1beta1.CustomResourceValidation{
			OpenAPIV3Schema: &v1beta1.JSONSchemaProps{
				Type: "object",
				Properties: map[string]v1beta1.JSONSchemaProps{
					"spec": v1beta1.JSONSchemaProps{
						Type: "object",
						Properties: map[string]v1beta1.JSONSchemaProps{
							"fluentconf": v1beta1.JSONSchemaProps{
								Type: "string",
							},
						},
					},
				},
			},
		},
		Scope: v1beta1.NamespaceScoped,
		Versions: []v1beta1.CustomResourceDefinitionVersion{
			v1beta1.CustomResourceDefinitionVersion{
				Name:    "v1beta1",
				Served:  true,
				Storage: true,
			},
		},
	},
}

type v1beta1Manager struct {
	clientset *clientset.Clientset
}

func (m *v1beta1Manager) ApplyCRD() error {
	if _, err := m.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(&legacyFluentdConfigCRD); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	return nil
}

func (m *v1beta1Manager) CheckCRD() (bool, error) {
	crd, err := m.clientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(m.GetCRDName(), metav1.GetOptions{})
	if err != nil {
		return false, err
	}

	for _, cond := range crd.Status.Conditions {
		if cond.Type == v1beta1.Established && cond.Status == v1beta1.ConditionTrue {
			return true, nil
		}
	}
	return false, nil
}

func (m *v1beta1Manager) GetCRDName() string {
	return legacyFluentdConfigCRD.ObjectMeta.Name
}
