package kubedatasource

import (
	"time"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"

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

func NewMigrationModeDS(cfg *config.Config, kubeCfg *rest.Config, factory informers.SharedInformerFactory, updateChan chan time.Time) (*MigrationModeDS, error) {
	fdKubeDS, err := NewFluentdConfigDS(cfg, kubeCfg, updateChan)
	if err != nil {
		return nil, err
	}

	cmKubeDS, err := NewConfigMapDS(cfg, factory, updateChan)
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
func (m *MigrationModeDS) GetFluentdConfig(namespace string) (string, error) {
	fdConfigs, err := m.fdKubeDS.GetFluentdConfig(namespace)
	if err != nil {
		return "", err
	}

	cmConfigs, err := m.cmKubeDS.GetFluentdConfig(namespace)
	if err != nil {
		return "", err
	}

	return cmConfigs + "\n" + fdConfigs, nil
}
