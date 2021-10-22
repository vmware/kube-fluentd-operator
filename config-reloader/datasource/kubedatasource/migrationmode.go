package kubedatasource

import (
	"context"
	"time"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	kfoListersV1beta1 "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/listers/logs.vdp.vmware.com/v1beta1"

	"k8s.io/client-go/informers"
	"k8s.io/client-go/rest"
)

// MigrationModeDS is an abstraction providing a virtual KubeDS instance
// that internally uses both the fluentdConfig KubeDS and the configMap KubeDS
// This abstraction allows both KubeDS to be used together
type MigrationModeDS struct {
	fdKubeDS KubeDS
	cmKubeDS KubeDS
}

func NewMigrationModeDS(ctx context.Context, cfg *config.Config, kubeCfg *rest.Config, factory informers.SharedInformerFactory, updateChan chan time.Time) (*MigrationModeDS, error) {
	fdKubeDS, err := NewFluentdConfigDS(ctx, cfg, kubeCfg, updateChan)
	if err != nil {
		return nil, err
	}

	cmKubeDS, err := NewConfigMapDS(ctx, cfg, factory, updateChan)
	if err != nil {
		return nil, err
	}

	return &MigrationModeDS{
		fdKubeDS: fdKubeDS,
		cmKubeDS: cmKubeDS,
	}, nil
}

// IsReady returns a boolean specifying whether the MigrationModeDS is ready
func (m *MigrationModeDS) IsReady() bool {
	return m.fdKubeDS.IsReady() && m.cmKubeDS.IsReady()
}

// GetFluentdConfig returns the fluentd configs for the given ns extracted
// by the two KubeDS and concatenated together
func (m *MigrationModeDS) GetFluentdConfig(ctx context.Context, namespace string) (string, error) {
	fdConfigs, err := m.fdKubeDS.GetFluentdConfig(ctx, namespace)
	if err != nil {
		return "", err
	}

	cmConfigs, err := m.cmKubeDS.GetFluentdConfig(ctx, namespace)
	if err != nil {
		return "", err
	}

	return cmConfigs + "\n" + fdConfigs, nil
}

// GetFdlist return nil for this mode because it does not use CRDs:
func (m *MigrationModeDS) GetFdlist() kfoListersV1beta1.FluentdConfigLister {
	return nil
}
