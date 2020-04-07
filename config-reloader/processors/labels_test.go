// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func TestLabelsParseOk(t *testing.T) {
	inputs := map[string]map[string]string{
		"$labels(a=b,,,)":                  {"a": "b"},
		"$labels(a=1, b=2)":                {"a": "1", "b": "2"},
		"$labels(x=y,b=1)":                 {"b": "1", "x": "y"},
		"$labels(x=1, b = 1)":              {"b": "1", "x": "1"},
		"$labels(x=1, a=)":                 {"a": "", "x": "1"},
		"$labels(hello/world=ok, a=value)": {"hello/world": "ok", "a": "value"},
		"$labels(x=1, _container=main)":    {"_container": "main", "x": "1"},
	}

	for tag, result := range inputs {
		processed, err := parseTagToLabels(tag)
		assert.Nil(t, err, "Got an error instead: %+v", err)
		assert.Equal(t, result, processed)
	}
}

func TestSafeLabel(t *testing.T) {
	// empty string is a valid label value
	assert.Equal(t, "_", safeLabelValue(""))

	assert.Equal(t, "abc", safeLabelValue("abc"))
	assert.Equal(t, "_abc_", safeLabelValue("-abc-"))
	assert.Equal(t, "abc___", safeLabelValue("abc..."))
	assert.Equal(t, "abc_def", safeLabelValue("abc.def"))
}

func TestLabelsParseNotOk(t *testing.T) {
	inputs := []string{
		"$labels",
		"$labels()",
		"$labels(=)",
		"$labels(=f)",
		"$labels(.=*)",
		"$labels(a=.)",
		"$labels(a==1)",
		"$labels(-a=sfd)",
		"$labels(a=-sfd)",
		"$labels(a*=hello)",
		"$labels(a=*)",
		"$labels(a=1, =2)",
		"$labels(_container=)", // empty container name
	}

	for _, tag := range inputs {
		res, err := parseTagToLabels(tag)
		assert.NotNil(t, err, "Got this instead for %s: %+v", tag, res)

	}
}

func TestLabelNoLabels(t *testing.T) {
	s := `
<filter **>
  @type parse
</filter>

<match **>
  @type logzio
</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	fragment, err = Process(fragment, ctx, &expandLabelsMacroState{})
	fmt.Printf("Processed:\n%s\n", fragment)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(fragment))
}

func TestLabelWithLabels(t *testing.T) {
	s := `
<filter $labels(app=grafana, release=rel)>
  @type parse
</filter>

<match $labels(app=grafana, release=rel)>
  @type logzio
</match>

<match $labels(app=prom, heritage=helm.12)>
  @type logzio
</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	fragment, err = Process(fragment, ctx, &expandLabelsMacroState{})
	assert.Nil(t, err)

	fmt.Printf("Processed:\n%s\n", fragment)

	assert.Equal(t, 6, len(fragment))

	if dir := fragment[0]; true {
		assert.Equal(t, "filter", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*", dir.Tag)
		assert.Equal(t, "record_transformer", dir.Type())
	}

	if dir := fragment[1]; true {
		assert.Equal(t, "match", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*", dir.Tag)
		assert.Equal(t, "rewrite_tag_filter", dir.Type())
	}

	if dir := fragment[2]; true {
		assert.Equal(t, "filter", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*.**", dir.Tag)
		assert.Equal(t, "record_transformer", dir.Type())
		assert.Equal(t, "kubernetes_pod_label_values", dir.Param("remove_keys"))
	}

	if dir := fragment[3]; true {
		assert.Equal(t, "filter", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*._labels.grafana.*.rel", dir.Tag)
	}

	if dir := fragment[4]; true {
		assert.Equal(t, "match", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*._labels.grafana.*.rel", dir.Tag)
	}

	if dir := fragment[5]; true {
		assert.Equal(t, "match", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*._labels.prom.helm_12.*", dir.Tag)
	}
}

func TestLabelWithLabelsAndElse(t *testing.T) {
	s := `
<filter $labels(app=grafana, release=rel)>
  @type parse
</filter>

<match $labels(app=grafana, release=rel)>
  @type logzio
</match>

<match $labels(app=prom, heritage=Helm)>
  @type logzio
</match>
<filter kube.monitoring.**>
  @type null
</filter>

<match **>
  @type null
</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
		AllowTagExpansion: true,
	}

	fragment, err = Process(fragment, ctx, DefaultProcessors()...)
	assert.Nil(t, err)

	fmt.Printf("Processed:\n%s\n", fragment)
}

func TestLabelWithLabelsAndContainer(t *testing.T) {
	s := `
<filter $labels(app=grafana, _container=sidecar)>
  @type parse
</filter>

<match $labels(app=grafana, release=rel)>
  @type logzio
</match>

<match $labels(app=prom, heritage=helm.12)>
  @type logzio
</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	fragment, err = Process(fragment, ctx, &expandLabelsMacroState{})
	assert.Nil(t, err)

	fmt.Printf("Processed:\n%s\n", fragment)

	assert.Equal(t, 6, len(fragment))

	if dir := fragment[0]; true {
		assert.Equal(t, "filter", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*", dir.Tag)
		assert.Equal(t, "record_transformer", dir.Type())
		record := dir.Nested[0]
		assert.Equal(t, "record", record.Name)
		assert.Equal(t, `${record.dig('kubernetes','labels','app')&.gsub(/[.-]/, '_') || '_'}.${record.dig('kubernetes','labels','heritage')&.gsub(/[.-]/, '_') || '_'}.${record.dig('kubernetes','labels','release')&.gsub(/[.-]/, '_') || '_'}`, record.Param("kubernetes_pod_label_values"))
	}

	if dir := fragment[1]; true {
		assert.Equal(t, "match", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*", dir.Tag)
		assert.Equal(t, "rewrite_tag_filter", dir.Type())

		rule := dir.Nested[0]
		assert.Equal(t, "rule", rule.Name)
		assert.Equal(t, "kubernetes_pod_label_values", rule.Param("key"))
	}

	if dir := fragment[2]; true {
		assert.Equal(t, "filter", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*.**", dir.Tag)
		assert.Equal(t, "record_transformer", dir.Type())
		assert.Equal(t, "kubernetes_pod_label_values", dir.Param("remove_keys"))
	}

	if dir := fragment[3]; true {
		assert.Equal(t, "filter", dir.Name)
		assert.Equal(t, "kube.monitoring.*.sidecar._labels.grafana.*.*", dir.Tag)
	}

	if dir := fragment[4]; true {
		assert.Equal(t, "match", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*._labels.grafana.*.rel", dir.Tag)
	}

	if dir := fragment[5]; true {
		assert.Equal(t, "match", dir.Name)
		assert.Equal(t, "kube.monitoring.*.*._labels.prom.helm_12.*", dir.Tag)
	}
}
