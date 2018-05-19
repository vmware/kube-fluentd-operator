// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

func TestMountedFileRemovedAfterProcessing(t *testing.T) {
	s := `
<source>
  @type mounted-file
  path /hello/world
  labels app=spring-mvc  
</source>

<match **>
  @type logzio
</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original: %s", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
	}

	processor := &mountedFileState{}
	fragment, err = Process(fragment, ctx, processor)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(fragment))

	// assert.Equal(t, 1, len(processor.ContainerFiles))

	// cf := processor.ContainerFiles[0]
	// assert.Equal(t, "/hello/world", cf.Path)
	// assert.Equal(t, map[string]string{"app": "spring-mvc"}, cf.Labels)
}

func TestMountedFileCatchesMissingFile(t *testing.T) {
	missingPath := `
	<source>
	  @type mounted-file
	  labels app=spring-mvc  
	</source>
	
	<match **>
	  @type logzio
	</match
	`
	fragment, err := fluentd.ParseString(missingPath)
	assert.Nil(t, fragment)
	assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
}

func TestMountedFileCatchesEmptyLabels(t *testing.T) {
	missingPath := `
	<source>
	  @type mounted-file
	  labels 
	</source>
	
	<match **>
	  @type logzio
	</match
	`
	fragment, err := fluentd.ParseString(missingPath)
	assert.Nil(t, fragment)
	assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
}

func TestMountedFileCatchesMissingLabels(t *testing.T) {
	missingPath := `
	<source>
	  @type mounted-file
	  file /etc/hosts
	</source>
	
	<match **>
	  @type logzio
	</match
	`
	fragment, err := fluentd.ParseString(missingPath)
	assert.Nil(t, fragment)
	assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
}

func TestMatches(t *testing.T) {
	spec := &ContainerFile{
		Path: "/var/log/https.log",
	}
	mini := &datasource.MiniContainer{
		PodID:  "123",
		Name:   "container-name",
		Labels: map[string]string{"key": "value"},
	}

	assert.True(t, matches(spec, mini))

	spec.Labels = map[string]string{"_container": "hello"}
	assert.False(t, matches(spec, mini))

	spec.Labels = map[string]string{"_container": mini.Name}
	assert.True(t, matches(spec, mini))

	spec.Labels = map[string]string{"a": "a"}
	assert.False(t, matches(spec, mini))

	spec.Labels = map[string]string{"key": "value"}
	assert.True(t, matches(spec, mini))

	spec.Labels = map[string]string{"key": "value", "_container": "container-name"}
	assert.True(t, matches(spec, mini))

	spec.Labels = map[string]string{"a": "a", "key": "value", "_container": "container-name"}
	assert.False(t, matches(spec, mini))
}

func TestConvertToFragment(t *testing.T) {
	specC1 := &ContainerFile{
		Path:   "/var/log/redis.log",
		Labels: map[string]string{"key": "value", "_container": "container-name"},
	}

	c1 := &datasource.MiniContainer{
		PodID:   "123-id",
		PodName: "123",
		Name:    "container-name",
		Labels:  map[string]string{"key": "value"},
		HostMounts: []*datasource.Mount{
			{
				Path:       "/var/log",
				VolumeName: "logs",
			},
		},
	}

	specC2 := &ContainerFile{
		Path:   "/var/log/nginx.log",
		Labels: map[string]string{"app": "nginx"},
	}
	c2 := &datasource.MiniContainer{
		PodID:   "abc-id",
		PodName: "abc",
		Name:    "nginx",
		Labels:  map[string]string{"app": "nginx"},
		HostMounts: []*datasource.Mount{
			{
				Path:       "/var/log",
				VolumeName: "logs",
			},
			{
				Path:       "/var",
				VolumeName: "var",
			},
		},
	}

	ctx := &ProcessorContext{
		Namepsace:   "monitoring",
		KubeletRoot: "/kubelet-root",
		MiniContainers: []*datasource.MiniContainer{
			c1,
			c2,
		},
	}

	state := &mountedFileState{
		BaseProcessorState: BaseProcessorState{
			Context: ctx,
		},
	}

	result := state.convertToFragement(specC1)
	assert.Equal(t, 2, len(result))

	dir := result[0]

	assert.Equal(t, "source", dir.Name)
	assert.Equal(t, "tail", dir.Type())
	assert.Equal(t, "/kubelet-root/pods/123-id/volumes/kubernetes.io~empty-dir/logs/redis.log", dir.Param("path"))
	assert.Equal(t, "kube.monitoring.123.container-name", dir.Param("tag"))
	assert.Equal(t, "parse", dir.Nested[0].Name)
	assert.Equal(t, "none", dir.Nested[0].Type())

	mod := result[1]
	assert.Equal(t, "filter", mod.Name)
	assert.Equal(t, "record_modifier", mod.Type())

	result = state.convertToFragement(specC2)
	assert.Equal(t, 2, len(result))

	dir = result[0]

	assert.Equal(t, "source", dir.Name)
	assert.Equal(t, "tail", dir.Type())
	assert.Equal(t, "/kubelet-root/pods/abc-id/volumes/kubernetes.io~empty-dir/logs/nginx.log", dir.Param("path"))
	assert.Equal(t, "kube.monitoring.abc.nginx", dir.Param("tag"))

	mod = result[1]
	assert.Equal(t, "filter", mod.Name)
	assert.Equal(t, "record_modifier", mod.Type())
}

func TestProcessMountedFile(t *testing.T) {
	c1 := &datasource.MiniContainer{
		PodID:   "123-id",
		PodName: "123",
		Name:    "redis-main",
		Labels:  map[string]string{"app": "redis"},
		HostMounts: []*datasource.Mount{
			{
				Path:       "/var/log",
				VolumeName: "logs",
			},
		},
	}

	c2 := &datasource.MiniContainer{
		PodID:   "abc-id",
		PodName: "abc",
		Name:    "nginx-main",
		Labels:  map[string]string{"app": "nginx"},
		HostMounts: []*datasource.Mount{
			{
				Path:       "/var/log",
				VolumeName: "logs",
			},
			{
				Path:       "/var",
				VolumeName: "var",
			},
		},
	}

	ctx := &ProcessorContext{
		Namepsace:   "monitoring",
		KubeletRoot: "/kubelet-root",
		MiniContainers: []*datasource.MiniContainer{
			c1,
			c2,
		},
	}

	state := &mountedFileState{
		BaseProcessorState: BaseProcessorState{
			Context: ctx,
		},
	}

	s := `
	<source>
		@type mounted-file
		path /var/log/redis.log
		labels app=redis
	</source>

	<source>
		@type mounted-file
		path /var/log/nginx.log
		labels app=nginx, _container=nginx-main
	</source>

	<match **>
		@type null
	</match>
	`

	input, err := fluentd.ParseString(s)
	assert.Nil(t, err, "Must have parsed, instead got error %+v", err)

	prep, err := Prepare(input, ctx, state)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(prep))
	assert.Equal(t, "/kubelet-root/pods/123-id/volumes/kubernetes.io~empty-dir/logs/redis.log", prep[0].Param("path"))
	assert.Equal(t, "/kubelet-root/pods/abc-id/volumes/kubernetes.io~empty-dir/logs/nginx.log", prep[2].Param("path"))

	main, err := Process(input, ctx, state)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(main))
}
