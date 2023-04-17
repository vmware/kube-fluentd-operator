package datasource

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type testConfig struct {
	config         config.Config
	expectErr      bool
	expectedResult int
	namespaces     []string
	configmap      map[string]string
}

func TestGetNamespaces(t *testing.T) {
	assert := assert.New(t)
	configs := []testConfig{
		{
			config: config.Config{
				Datasource:         "default",
				LogLevel:           logrus.InfoLevel.String(),
				FluentdLogLevel:    "info",
				AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
				ID:                 "default",
			},
			expectErr:      false,
			expectedResult: 2,
			namespaces: []string{
				"test1",
				"test2",
			},
		},
		{
			config: config.Config{
				Datasource:         "default",
				LogLevel:           logrus.InfoLevel.String(),
				FluentdLogLevel:    "info",
				AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
				ID:                 "default",
			},
			expectErr:      false,
			expectedResult: 1,
			namespaces: []string{
				"test1",
			},
		},
	}
	for _, config := range configs {
		ctx := context.Background()
		if err := config.config.Validate(); err != nil {
			logrus.Fatalf("Config validation failed: %+v", err)
		}
		var namespaceObject []runtime.Object
		for _, ns := range config.namespaces {
			namespaceObject = append(namespaceObject,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: ns,
					},
				},
			)
		}
		clientset := testclient.NewSimpleClientset(namespaceObject...)

		ds, err := NewKubernetesInformerDatasource(ctx, &config.config, make(chan time.Time, 1), clientset)
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		namespaces, err := ds.GetNamespaces(ctx)
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		assert.Equal(len(namespaces), config.expectedResult)
		for index, ns := range config.namespaces {
			assert.Equal(namespaces[index].Name, ns)
		}
	}
}

func TestDiscoverNamespaces(t *testing.T) {
	assert := assert.New(t)
	configs := []testConfig{
		{
			config: config.Config{
				Datasource:         "default",
				LogLevel:           logrus.InfoLevel.String(),
				FluentdLogLevel:    "info",
				AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
				ID:                 "default",
			},
			expectErr:      false,
			expectedResult: 2,
			namespaces: []string{
				"test1",
				"test2",
			},
		},
		{
			config: config.Config{
				Datasource:           "default",
				LogLevel:             logrus.InfoLevel.String(),
				FluentdLogLevel:      "info",
				DefaultConfigmapName: "fluentd-config",
				AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
				ID:                   "default",
			},
			expectErr:      false,
			expectedResult: 2,
			namespaces: []string{
				"test1",
				"test2",
			},
			configmap: map[string]string{
				"configmap1":     "test1",
				"configmap2":     "test2",
				"fluentd-config": "test1",
			},
		},
	}
	for _, config := range configs {
		ctx := context.Background()
		if err := config.config.Validate(); err != nil {
			logrus.Fatalf("Config validation failed: %+v", err)
		}
		var k8sObject []runtime.Object

		for _, ns := range config.namespaces {
			k8sObject = append(k8sObject,
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: ns,
					},
				},
			)
		}
		for cm, ns := range config.configmap {
			k8sObject = append(k8sObject,
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cm,
						Namespace: ns,
					},
				},
			)
		}
		clientset := testclient.NewSimpleClientset(k8sObject...)
		factory := informers.NewSharedInformerFactory(clientset, 0)
		namespaceLister := factory.Core().V1().Namespaces().Lister()
		cmLister := factory.Core().V1().ConfigMaps().Lister()
		var ds = &kubeInformerConnection{
			client: clientset,
			cfg:    &config.config,
			nslist: namespaceLister,
			cmlist: cmLister,
		}

		for _, ns := range config.namespaces {
			factory.Core().V1().Namespaces().Informer().GetIndexer().Add(
				&corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{
						Name: ns,
					},
				},
			)
		}
		for cm, ns := range config.configmap {
			factory.Core().V1().ConfigMaps().Informer().GetIndexer().Add(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cm,
						Namespace: ns,
					},
				},
			)
		}
		namespaces, err := ds.discoverNamespaces(ctx)
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		assert.Equal(len(namespaces), config.expectedResult)
	}
}

func TestUpdateStatus(t *testing.T) {
	assert := assert.New(t)
	annotationValue := "Example annotation"
	namespace := "test-namespace"
	testCfg := &config.Config{
		Datasource:         "default",
		LogLevel:           logrus.InfoLevel.String(),
		FluentdLogLevel:    "info",
		AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
		AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
		ID:                 "default",
	}
	ctx := context.Background()
	if err := testCfg.Validate(); err != nil {
		logrus.Fatalf("Config validation failed: %+v", err)
	}
	clientset := testclient.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		},
	)
	ds, err := NewKubernetesInformerDatasource(ctx, testCfg, make(chan time.Time, 1), clientset)
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	ds.UpdateStatus(ctx, namespace, annotationValue)
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	assert.Equal(ns.Annotations[testCfg.AnnotStatus], annotationValue)
}
