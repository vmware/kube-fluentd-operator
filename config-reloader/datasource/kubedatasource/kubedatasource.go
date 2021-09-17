package kubedatasource

import (
	"context"
)

// KubeDS is an interface defining behavor for the Kubernetes Resources
// containing FluentD configurations
type KubeDS interface {
	GetFluentdConfig(ctx context.Context, namespace string) (string, error)
	IsReady() bool
}
