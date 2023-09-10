package datasource

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource/kubedatasource"
	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	testclient "k8s.io/client-go/kubernetes/fake"
)

type testConfig struct {
	config         config.Config
	expectErr      bool
	expectedResult int
	namespaces     []testNamespace
	configmap      map[string][]string
	configmapData  map[string]map[string]string
}

type testNamespace struct {
	name   string
	labels map[string]string
}

func importK8sObjects(factory informers.SharedInformerFactory, namespaces []testNamespace, configmaps map[string][]string, configmapData map[string]map[string]string) {
	for _, ns := range namespaces {
		factory.Core().V1().Namespaces().Informer().GetIndexer().Add(
			&corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:   ns.name,
					Labels: ns.labels,
				},
			},
		)
	}
	for ns, cms := range configmaps {
		for _, cm := range cms {
			factory.Core().V1().ConfigMaps().Informer().GetIndexer().Add(
				&corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cm,
						Namespace: ns,
					},
					Data: configmapData[ns],
				},
			)
		}
	}
}

// UnitTest for GetNamespaces while using configmap mode
func TestGetNamespaces(t *testing.T) {
	assert := assert.New(t)
	configs := []testConfig{
		{
			//TestCase: One Namespace without any fluentd config. Result empty
			config: config.Config{
				Datasource:         "default",
				AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
				ID:                 "default",
			},
			expectErr:      false,
			expectedResult: 0,
			namespaces: []testNamespace{
				{
					name: "test1",
				},
			},
		},
		{
			//TestCase: Two Namespaces, one without cm and one with empty fluentd config. Result empty
			config: config.Config{
				Datasource:         "default",
				AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
				ID:                 "default",
			},
			expectErr:      false,
			expectedResult: 0,
			namespaces: []testNamespace{
				{
					name: "test-empty",
				},
				{
					name: "test-empty-2",
				},
			},
			configmap: map[string][]string{
				"test-empty":   {"fluentd-config", "my-configmap1"},
				"test-empty-2": {"my-configmap2"},
			},
		},
		{
			//TestCase: Two Namespaces, one fluentd config. Result 1
			config: config.Config{
				Datasource:           "default",
				DefaultConfigmapName: "fluentd-config",
				AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
				ID:                   "default",
			},
			expectErr:      false,
			expectedResult: 1,
			namespaces: []testNamespace{
				{
					name: "test-not-empty",
				},
				{
					name: "test-not-empty-2",
				},
			},
			configmap: map[string][]string{
				"test-not-empty": {"fluentd-config"},
			},
			configmapData: map[string]map[string]string{"test-not-empty": {"fluent.conf": "<match **>\n@type stdout\n</match>"}},
		},
		{
			//TestCase: Two Namespaces, one fluentd config wrong. Result 1
			config: config.Config{
				Datasource:           "default",
				DefaultConfigmapName: "fluentd-config",
				AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
				ID:                   "default",
			},
			expectErr:      false,
			expectedResult: 1,
			namespaces: []testNamespace{
				{
					name: "test-not-empty",
				},
				{
					name: "test-not-empty-2",
				},
			},
			configmap: map[string][]string{
				"test-not-empty":   {"fluentd-config"},
				"test-not-empty-2": {"fluentd-config"},
			},
			configmapData: map[string]map[string]string{
				"test-not-empty":   {"fluent.conf": "<match **>\n@type stdout\n</match>"},
				"test-not-empty-2": {"fluent.conf": "<filter **>\n></filter><match **>\n@type stdout\n</match>"},
			},
		},
	}
	for _, config := range configs {
		// Prepare TestCase
		ctx := context.Background()
		clientset := testclient.NewSimpleClientset()
		factory := informers.NewSharedInformerFactory(clientset, 0)
		kubeds, _ := kubedatasource.NewConfigMapDS(ctx, &config.config, factory, make(chan time.Time, 1))
		var ds = &kubeInformerConnection{
			client:        clientset,
			cfg:           &config.config,
			nslist:        factory.Core().V1().Namespaces().Lister(),
			cmlist:        factory.Core().V1().ConfigMaps().Lister(),
			podlist:       factory.Core().V1().Pods().Lister(),
			kubeds:        kubeds,
			mountedLabels: make(map[string][]map[string]string),
		}
		importK8sObjects(factory, config.namespaces, config.configmap, config.configmapData)
		// Run Test GetNamespace
		namespaces, err := ds.GetNamespaces(ctx)
		// Check Test Result
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		assert.Equal(config.expectedResult, len(namespaces))
		for index, ns := range namespaces {
			assert.Equal(config.namespaces[index].name, ns.Name)
		}
	}
}

func TestDiscoverNamespaces(t *testing.T) {
	assert := assert.New(t)
	configs := []testConfig{
		{
			//TestCase: Two Namespaces, emptu DefaultConfigmapName. Result 2
			config: config.Config{
				Datasource:         "default",
				AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
				ID:                 "default",
			},
			expectErr:      false,
			expectedResult: 2,
			namespaces: []testNamespace{
				{
					name: "test1",
				},
				{
					name: "test2",
				},
			},
		},
		{
			//TestCase: Two Namespaces, use DefaultConfigmapName. Result 1
			config: config.Config{
				Datasource:           "default",
				DefaultConfigmapName: "fluentd-config",
				AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
				ID:                   "default",
			},
			expectErr:      false,
			expectedResult: 1,
			namespaces: []testNamespace{
				{
					name: "test1",
				},
				{
					name: "test2",
				},
			},
			configmap: map[string][]string{
				"test1": {"configmap1", "fluentd-config"},
				"test2": {"configmap2"},
			},
		},
		{
			//TestCase: Two Namespaces with different labels, use DefaultConfigmapName. Result 2
			config: config.Config{
				Datasource:           "default",
				DefaultConfigmapName: "fluentd-config",
				AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
				ID:                   "default",
				NamespaceSelector:    "key=value",
			},
			expectErr:      false,
			expectedResult: 2,
			namespaces: []testNamespace{
				{
					name: "test1",
					labels: map[string]string{
						"key1": "value1",
						"key":  "value",
					},
				},
				{
					name: "test2",
					labels: map[string]string{
						"key2": "value2",
						"key":  "value",
					},
				},
			},
			configmap: map[string][]string{
				"test1": {"configmap1", "fluentd-config"},
				"test2": {"configmap2"},
			},
		},
		{
			//TestCase: Two Namespaces with different labels, use DefaultConfigmapName. Result 1
			config: config.Config{
				Datasource:           "default",
				DefaultConfigmapName: "fluentd-config",
				AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
				AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
				ID:                   "default",
				NamespaceSelector:    "key=value,key1=value1",
			},
			expectErr:      false,
			expectedResult: 1,
			namespaces: []testNamespace{
				{
					name: "test1",
					labels: map[string]string{
						"key1": "value1",
						"key":  "value",
					},
				},
				{
					name: "test2",
					labels: map[string]string{
						"key2": "value2",
						"key":  "value",
					},
				},
			},
			configmap: map[string][]string{
				"test1": {"configmap1", "fluentd-config"},
				"test2": {"configmap2"},
			},
		},
	}
	for _, config := range configs {
		// Prepare TestCase
		ctx := context.Background()
		clientset := testclient.NewSimpleClientset()
		factory := informers.NewSharedInformerFactory(clientset, 0)
		var ds = &kubeInformerConnection{
			client: clientset,
			cfg:    &config.config,
			nslist: factory.Core().V1().Namespaces().Lister(),
			cmlist: factory.Core().V1().ConfigMaps().Lister(),
		}
		importK8sObjects(factory, config.namespaces, config.configmap, config.configmapData)
		// Run Test discoverNamespaces
		namespaces, err := ds.discoverNamespaces(ctx)
		// Check Test Result
		if err != nil {
			logrus.Fatalf(err.Error())
		}
		assert.Equal(config.expectedResult, len(namespaces))
	}
}

// Test the updateStatus config creating a namespace and check label and value
func TestUpdateStatus(t *testing.T) {
	assert := assert.New(t)
	annotationValue := "Example annotation"
	namespace := "test-namespace"
	testCfg := &config.Config{
		Datasource:         "default",
		AnnotConfigmapName: "logging.csp.vmware.com/fluentd-configmap",
		AnnotStatus:        "logging.csp.vmware.com/fluentd-status",
		ID:                 "default",
	}
	ctx := context.Background()
	clientset := testclient.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		},
	)
	factory := informers.NewSharedInformerFactory(clientset, 0)
	var ds = &kubeInformerConnection{
		client: clientset,
		cfg:    testCfg,
		nslist: factory.Core().V1().Namespaces().Lister(),
		cmlist: factory.Core().V1().ConfigMaps().Lister(),
	}

	ds.UpdateStatus(ctx, namespace, annotationValue)
	ns, err := clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
	if err != nil {
		logrus.Fatalf("Unable to read namespace in cluster: %+v", err)
	}
	assert.Equal(ns.Annotations[testCfg.AnnotStatus], annotationValue)
}
