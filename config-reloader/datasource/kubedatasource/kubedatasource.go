package kubedatasource

// KubeDS is an interface defining behavor for the Kubernetes Resources
// containing FluentD configurations
type KubeDS interface {
	GetFluentdConfig(namespace string) (string, error)
	IsReady() bool
}
