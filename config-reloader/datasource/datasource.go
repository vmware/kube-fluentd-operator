// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package datasource

import (
	"sort"
	core "k8s.io/api/core/v1"
)

const (
	entryName = "fluent.conf"
)

type Mount struct {
	Path       string
	VolumeName string
}

// MiniContainer container subset with the parent pod's metadata
type MiniContainer struct {
	// the pod id
	PodID   string
	PodName string

	// pod labels
	Labels map[string]string

	// container name
	Name string
	// only the emptyDir mounts, never empty, sorted by len(Path), descending
	HostMounts []*Mount

	NodeName string
}

// NamespaceConfig holds all relevant data for a namespace
type NamespaceConfig struct {
	Name               string
	FluentdConfig      string
	PreviousConfigHash string
	IsKnownFromBefore  bool
	MiniContainers     []*MiniContainer
	Labels             map[string]string
}

// StatusUpdater sets an error description on the namespace
// in case configuration cannot be applied or an empty string otherwise
type StatusUpdater interface {
	UpdateStatus(namespace string, status string)
}

// Datasource reads data from k8s
type Datasource interface {
	StatusUpdater
	GetNamespaces() ([]*NamespaceConfig, error)
	WriteCurrentConfigHash(namespace string, hash string)
}

type byLength []*Mount

func (s byLength) Len() int {
	return len(s)
}

func (s byLength) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s byLength) Less(i, j int) bool {
	return len(s[i].Path) > len(s[j].Path)
}

func convertPodToMinis(resp *core.PodList) []*MiniContainer {
	var res []*MiniContainer

	for _, pod := range resp.Items {
		for _, cont := range pod.Spec.Containers {
			mini := &MiniContainer{
				PodID:    string(pod.UID),
				PodName:  pod.Name,
				Labels:   pod.Labels,
				Name:     cont.Name,
				NodeName: pod.Spec.NodeName,
			}

			for _, vm := range cont.VolumeMounts {
				m := makeVolume(pod.Spec.Volumes, &vm)
				if m != nil {
					mini.HostMounts = append(mini.HostMounts, m)
				}
			}

			if len(mini.HostMounts) > 0 {
				sort.Sort(byLength(mini.HostMounts))
				res = append(res, mini)
			}
		}
	}
	return res
}

func makeVolume(volumes []core.Volume, volumeMount *core.VolumeMount) *Mount {
	for _, v := range volumes {
		if v.Name == volumeMount.Name && v.EmptyDir != nil {
			return &Mount{
				VolumeName: v.Name,
				Path:       volumeMount.MountPath,
			}
		}
	}
	return nil
}
