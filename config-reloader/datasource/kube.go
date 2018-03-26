// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package datasource

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"

	"github.com/sirupsen/logrus"
	core "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	entryName = "fluent.conf"
)

type kubeConnection struct {
	client kubernetes.Interface
	hashes map[string]string
	cfg    *config.Config
}

func (d *kubeConnection) readConfig(namespace string, configMapName string) (string, error) {
	opts := meta_v1.GetOptions{}
	configMap, err := d.client.Core().ConfigMaps(namespace).Get(configMapName, opts)

	if err != nil {
		return "", err
	}

	contents, ok := configMap.Data[entryName]
	if !ok {
		return "", fmt.Errorf("cannot find entry %s in config map %s/%s", entryName, namespace, configMapName)
	}

	return contents, nil
}

func (d *kubeConnection) unconfiguredNamespace(ns string) *NamespaceConfig {
	return &NamespaceConfig{
		Name:               ns,
		FluentdConfig:      "",
		PreviousConfigHash: d.hashes[ns],
	}
}

func (d *kubeConnection) GetNamespaces() ([]*NamespaceConfig, error) {
	resp, err := d.client.CoreV1().Namespaces().List(meta_v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := []*NamespaceConfig{}
	for _, item := range resp.Items {
		configMapName := item.Annotations[d.cfg.AnnotConfigmapName]
		if configMapName == "" {
			logrus.Debugf("Will not process namespace '%s': not annotated with '%s'", item.Name, d.cfg.AnnotConfigmapName)
			// namespace not annotated
			result = append(result, d.unconfiguredNamespace(item.Name))
			continue
		}

		contents, err := d.readConfig(item.Name, configMapName)
		if err != nil {
			logrus.Debugf("Will not process namespace '%s': %+v", item.Name, err)
			result = append(result, d.unconfiguredNamespace(item.Name))
			continue
		}

		logrus.Debugf("Processing namespace '%s' using configmap '%s'", item.Name, configMapName)

		obj := &NamespaceConfig{
			Name:               item.Name,
			FluentdConfig:      contents,
			PreviousConfigHash: d.hashes[item.Name],
			IsKnownFromBefore:  true,
		}

		result = append(result, obj)
	}

	return result, nil
}

func (d *kubeConnection) WriteCurrentConfigHash(namespace string, hash string) {
	d.hashes[namespace] = hash
}

func (d *kubeConnection) UpdateStatus(namespace string, status string) {
	patch := &core.Namespace{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: namespace,
			Annotations: map[string]string{
				d.cfg.AnnotStatus: status,
			},
		},
	}

	body, _ := json.Marshal(&patch)
	_, err := d.client.CoreV1().Namespaces().Patch(namespace, types.MergePatchType, body)

	logrus.Debugf("Saving status: %+v, %+v", patch, err)
	if err != nil {
		logrus.Infof("Cannot set error status of %s: %v", namespace, err)
	}
}

func NewKubernetesDatasource(cfg *config.Config) (Datasource, error) {
	kubeConfig := cfg.KubeConfig
	if cfg.KubeConfig == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			kubeConfig = clientcmd.RecommendedHomeFile
		}
	}

	config, err := clientcmd.BuildConfigFromFlags(cfg.Master, kubeConfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Connected to cluster at %s", config.Host)

	return &kubeConnection{
		client: client,
		hashes: make(map[string]string),
		cfg:    cfg,
	}, nil
}
