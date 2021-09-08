module github.com/vmware/kube-fluentd-operator/config-reloader

go 1.16

require (
	github.com/alecthomas/kingpin v2.2.6+incompatible
	github.com/prometheus/client_golang v1.11.0
	github.com/sirupsen/logrus v1.7.0
	github.com/stretchr/testify v1.6.1
	k8s.io/api v0.21.4
  k8s.io/apiextensions-apiserver v0.21.4
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
)
