package kubedatasource

import (
	"context"
	kfoListersV1beta1 "github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource/fluentdconfig/client/listers/logs.vdp.vmware.com/v1beta1"
)

// KubeDS is an interface defining behavor for the Kubernetes Resources
// containing FluentD configurations
type KubeDS interface {
	GetFluentdConfig(ctx context.Context, namespace string) (string, error)
	IsReady() bool
	GetFdlist() kfoListersV1beta1.FluentdConfigLister
}
