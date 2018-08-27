// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package datasource

import (
	"encoding/json"
	"os"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"

	"github.com/sirupsen/logrus"
	core "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"fmt"
)

type kubeMultimapConnection struct {
	client kubernetes.Interface
	hashes map[string]string
	cfg    *config.Config
}

func (d *kubeMultimapConnection) readConfig(namespace string, maps *core.ConfigMapList) (string, error) {
	contents := ""
	for _, configMap := range maps.Items {
		if _, ok := configMap.Annotations[d.cfg.AnnotConfigmapName]; ok {
			mapData, ok := configMap.Data[entryName]
			if ok {
				contents = fmt.Sprintf("%s\n%s", contents, mapData)
				logrus.Debugf("loaded config data from config map %s/%s", namespace, configMap.Name)
			} else {
				logrus.Warnf("cannot find entry %s in config map %s/%s", entryName, namespace, configMap.Name)
			}
		}
	}

	return contents, nil
}

func (d *kubeMultimapConnection) unconfiguredNamespace(ns string) *NamespaceConfig {
	return &NamespaceConfig{
		Name:               ns,
		FluentdConfig:      "",
		PreviousConfigHash: d.hashes[ns],
	}
}

func (d *kubeMultimapConnection) GetNamespaces() ([]*NamespaceConfig, error) {
	resp, err := d.client.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var result []*NamespaceConfig
	for _, item := range resp.Items {
		maps, err := d.client.CoreV1().ConfigMaps(item.Name).List(metav1.ListOptions{})
		if err != nil {
			logrus.Debugf("will not process namespace '%s': %+v", item.Name, err)
			result = append(result, d.unconfiguredNamespace(item.Name))
			continue
		}

		if !d.needsProcessing(item.Name, maps) {
			continue
		}

		contents, err := d.readConfig(item.Name, maps)
		if err != nil {
			logrus.Debugf("will not process namespace '%s': %+v", item.Name, err)
			result = append(result, d.unconfiguredNamespace(item.Name))
			continue
		}

		logrus.Debugf("processing namespace '%s'", item.Name)

		obj := &NamespaceConfig{
			Name:               item.Name,
			FluentdConfig:      contents,
			PreviousConfigHash: d.hashes[item.Name],
			IsKnownFromBefore:  true,
			Labels:             item.Labels,
		}

		resp, err := d.client.CoreV1().Pods(item.Name).List(metav1.ListOptions{})
		if err == nil {
			obj.MiniContainers = convertPodToMinis(resp)
		} else {
			logrus.Infof("Cannot read pods in namespace '%s'", item.Name)
		}

		result = append(result, obj)
	}

	return result, nil
}

func (d *kubeMultimapConnection) needsProcessing(ns string, maps *core.ConfigMapList) bool {
	if len(d.cfg.Namespaces) == 0 {
		if d.containsProcessableMap(maps) {
			return true
		}
		logrus.Debugf("ignoring namespace '%s' because it doesn't contain any processable map", ns)
		return false
	}

	for _, item := range d.cfg.Namespaces {
		if item == ns {
			if d.containsProcessableMap(maps) {
				return true
			}
			logrus.Debugf("ignoring namespace '%s' because it doesn't contain any processable map", ns)
			return false
		}
	}

	logrus.Debugf("ignoring namespace '%s' because of --namespaces flag", ns)
	return false
}

func (d *kubeMultimapConnection) containsProcessableMap(maps *core.ConfigMapList) (bool) {
	for _, configMap := range maps.Items {
		if _, ok := configMap.Annotations[d.cfg.AnnotConfigmapName]; ok {
			return true
		}
	}
	return false
}


func (d *kubeMultimapConnection) WriteCurrentConfigHash(namespace string, hash string) {
	d.hashes[namespace] = hash
}

func (d *kubeMultimapConnection) UpdateStatus(namespace string, status string) {
	patch := &core.Namespace{
		ObjectMeta: metav1.ObjectMeta{
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

func NewKubernetesMultimapDatasource(cfg *config.Config) (Datasource, error) {
	kubeConfig := cfg.KubeConfig
	if cfg.KubeConfig == "" {
		if _, err := os.Stat(clientcmd.RecommendedHomeFile); err == nil {
			kubeConfig = clientcmd.RecommendedHomeFile
		}
	}

	kubeCfg, err := clientcmd.BuildConfigFromFlags(cfg.Master, kubeConfig)
	if err != nil {
		return nil, err
	}

	client, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		return nil, err
	}

	logrus.Infof("Connected to cluster at %s", kubeCfg.Host)

	return &kubeMultimapConnection{
		client: client,
		hashes: make(map[string]string),
		cfg:    cfg,
	}, nil
}
