// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

const mountedFileSourceType = "mounted-file"

type ContainerFile struct {
	Labels map[string]string
	Path   string
}

type MountedFileState struct {
	BaseProcessorState
	ContainerFiles []*ContainerFile
}

func (state *MountedFileState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	res := fluentd.Fragment{}

	for _, frag := range input {
		if frag.Name == "source" && frag.Type() == mountedFileSourceType {
			// store the mounted file info in the context, remove from the final config

			paramLabels := frag.Param("labels")
			if paramLabels == "" {
				return nil, fmt.Errorf("'labels' is required when using @type %s", mountedFileSourceType)
			}
			paramLabels = util.TrimTrailingComment(paramLabels)

			labels, err := parseTagToLabels(fmt.Sprintf("$labels(%s)", paramLabels))
			if err != nil {
				return nil, err
			}
			cf := &ContainerFile{}
			cf.Labels = labels

			paramPath := frag.Param("path")
			if paramPath == "" {
				return nil, fmt.Errorf("'path' is required when using @type %s", mountedFileSourceType)
			}
			cf.Path = paramPath

			state.ContainerFiles = append(state.ContainerFiles, cf)
		} else {
			res = append(res, frag)
		}
	}

	return res, nil
}
