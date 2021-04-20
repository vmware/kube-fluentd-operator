// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"bytes"
	"fmt"
	"path"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

const mountedFileSourceType = "mounted-file"

// ContainerFile stores parsed info from a <source> @type mounted-file...
type ContainerFile struct {
	Labels      map[string]string
	AddedLabels map[string]string
	Path        string
	Parse       *fluentd.Directive
}

type mountedFileState struct {
	BaseProcessorState
}

func isRelevant(frag *fluentd.Directive) bool {
	return frag.Name == "source" && frag.Type() == mountedFileSourceType
}

func (state *mountedFileState) Prepare(input fluentd.Fragment) (fluentd.Fragment, error) {
	res := fluentd.Fragment{}

	for _, frag := range input {
		if isRelevant(frag) {
			paramLabels := frag.Param("labels")
			if paramLabels == "" {
				return nil, fmt.Errorf("'labels' is required when using @type %s", mountedFileSourceType)
			}
			paramLabels = util.TrimTrailingComment(paramLabels)

			labels, err := parseTagToLabels(fmt.Sprintf("$labels(%s)", paramLabels))
			if err != nil {
				return nil, err
			}

			paramAddedLabels := frag.Param("add_labels")
			paramAddedLabels = util.TrimTrailingComment(paramAddedLabels)
			var addedLabels map[string]string
			if paramAddedLabels != "" {
				// no added labels is just fine
				addedLabels, err = parseTagToLabels(fmt.Sprintf("$labels(%s)", paramAddedLabels))
				if err != nil {
					return nil, err
				}
			}

			cf := &ContainerFile{}
			cf.Labels = labels
			cf.AddedLabels = addedLabels

			paramPath := frag.Param("path")
			if paramPath == "" {
				return nil, fmt.Errorf("'path' is required when using @type %s", mountedFileSourceType)
			}
			cf.Path = paramPath

			if len(frag.Nested) == 1 {
				cf.Parse = frag.Nested[0]
			} else if len(frag.Nested) >= 2 {
				return nil, fmt.Errorf("One or zero <parse> directives required when using @type %s, found %d", mountedFileSourceType, len(frag.Nested))
			}

			newFrag := state.convertToFragement(cf)
			if newFrag != nil {
				res = append(res, newFrag...)
			}
		}
	}

	return res, nil
}

func matches(spec *ContainerFile, mini *datasource.MiniContainer) bool {
	for k, v := range spec.Labels {
		contValue := mini.Labels[k]
		if k == "_container" {
			contValue = mini.Name
		}

		if v != contValue {
			return false
		}
	}

	return true
}

func (state *mountedFileState) convertToFragement(cf *ContainerFile) fluentd.Fragment {
	res := fluentd.Fragment{}
	for _, mc := range state.Context.MiniContainers {
		if matches(cf, mc) {
			dir := &fluentd.Directive{
				Name:   "source",
				Params: fluentd.Params{},
			}

			for _, hm := range mc.HostMounts {
				if !strings.HasPrefix(cf.Path, hm.Path) {
					// misconfiguration??
					continue
				}
				dir.SetParam("@type", "tail")

				hostPath := state.makeHostPath(cf, hm, mc)
				pos := util.Hash(state.Context.DeploymentID, fmt.Sprintf("%s-%s-%s", mc.PodID, mc.Name, hostPath))
				tag := fmt.Sprintf("kube.%s.%s.%s-%s", state.Context.Namespace, mc.PodName, mc.Name, pos)
				dir.SetParam("path", hostPath)
				dir.SetParam("read_from_head", "true")
				dir.SetParam("tag", tag)

				dir.SetParam("pos_file", fmt.Sprintf("/var/log/kfotail-%s.pos", pos))

				if cf.Parse != nil {
					dir.Nested = []*fluentd.Directive{
						cf.Parse,
					}
				} else {
					dir.Nested = []*fluentd.Directive{
						makeDefaultParseDirective(),
					}
				}
				res = append(res, dir, state.makeAttachK8sMetadataDirective(tag, mc, cf))
				break
			}
		}
	}

	return res
}

func (state *mountedFileState) makeAttachK8sMetadataDirective(tag string, mc *datasource.MiniContainer, cf *ContainerFile) *fluentd.Directive {
	res := &fluentd.Directive{
		Name:   "filter",
		Tag:    tag,
		Params: fluentd.Params{},
		Nested: []*fluentd.Directive{
			{
				Name:   "record",
				Params: fluentd.Params{},
			},
		},
	}

	res.SetParam("@type", "record_modifier")
	res.SetParam("remove_keys", "dummy_")

	buf := &bytes.Buffer{}
	fmt.Fprintf(buf, "record['stream']='%s'; ", cf.Path)
	fmt.Fprintf(buf, "record['kubernetes']=%s; ", util.ToRubyMapLiteral(map[string]string{
		"container_name":  mc.Name,
		"container_image": mc.Image,
		"namespace_name":  state.Context.Namespace,
		"pod_name":        mc.PodName,
		"pod_id":          mc.PodID,
		"host":            mc.NodeName,
	}))

	fmt.Fprintf(buf, "record['docker']=%s; ", util.ToRubyMapLiteral(map[string]string{
		"container_id": mc.ContainerID,
	}))

	fmt.Fprintf(buf, "record['container_info']='%s'; ", util.Hash(mc.PodID, cf.Path))
	fmt.Fprintf(buf, "record['kubernetes']['labels']=%s; ", util.ToRubyMapLiteral(mergeMaps(mc.Labels, cf.AddedLabels)))
	fmt.Fprintf(buf, "record['kubernetes']['namespace_labels']=%s", util.ToRubyMapLiteral(state.Context.NamespaceLabels))

	res.Nested[0].SetParam("dummy_", fmt.Sprintf("${%s}", buf.String()))

	return res
}

func mergeMaps(base, more map[string]string) map[string]string {
	res := map[string]string{}

	for k, v := range base {
		res[k] = v
	}

	for k, v := range more {
		if _, ok := res[k]; !ok {
			// cannot override labels assigned from kubernetes
			res[k] = v
		}
	}

	return res
}

func makeDefaultParseDirective() *fluentd.Directive {
	res := &fluentd.Directive{
		Name:   "parse",
		Params: fluentd.ParamsFromKV("@type", "none"),
	}

	return res
}

func (state *mountedFileState) makeHostPath(cf *ContainerFile, hm *datasource.Mount, mc *datasource.MiniContainer) string {
	// var/lib/kubelet/pods/8e0f9442-41b5-11e8-a138-02b2be114bba/volumes/kubernetes.io~empty-dir/empty/hello.log
	volumentName := hm.VolumeName
	subPath := cf.Path[len(hm.Path):]
	return path.Join(state.Context.KubeletRoot, "pods", mc.PodID, "volumes", "kubernetes.io~empty-dir", volumentName, hm.SubPath, subPath)
}

func (state *mountedFileState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	res := fluentd.Fragment{}

	// just skip all non-relevant source directives
	for _, dir := range input {
		if !isRelevant(dir) {
			res = append(res, dir)
		}
	}

	return res, nil
}
