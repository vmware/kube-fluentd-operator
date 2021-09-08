package metrics

import (
	"fmt"
	"net"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	LabelTargetNamespace = "target_namespace"
)

var namespaceConfigStatus = prometheus.NewGaugeVec(prometheus.GaugeOpts{
	Namespace: "kube_fluentd_operator",
	Name:      "namespace_config_status",
	Help:      "Current validation status of fluentd configs in the namespace. Values are 0 (validation error) or 1 (validation successful)",
}, []string{LabelTargetNamespace})

// SetNamespaceConfigStatusMetric sets the current metric value for a given namespace
func SetNamespaceConfigStatusMetric(namespace string, valid bool) {
	var value float64
	if valid {
		value = 1
	}

	namespaceConfigStatus.With(prometheus.Labels{LabelTargetNamespace: namespace}).Set(value)
}

// DeleteNamespaceConfigStatusMetric deletes the metric value for a given namespace
func DeleteNamespaceConfigStatusMetric(namespace string) {
	namespaceConfigStatus.Delete(prometheus.Labels{LabelTargetNamespace: namespace})
}

// InitMetrics should be called to initialize metrics and start the HTTP handler
func InitMetrics(port int) error {
	if err := serveMetrics(port); err != nil {
		return fmt.Errorf("Failed to start metrics handler: %s", err)
	}

	registerMetrics()
	return nil
}

func registerMetrics() {
	prometheus.MustRegister(namespaceConfigStatus)
}

func serveMetrics(port int) error {
	ln, err := net.Listen("tcp", ":"+strconv.Itoa(port))
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	srv := &http.Server{Handler: mux}
	go func() {
		srv.Serve(ln)
	}()
	return nil
}
