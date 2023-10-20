// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/template"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

type expandLabelsMacroState struct {
	BaseProcessorState
}

var reSafe = regexp.MustCompile(`[.-]|^$`)

// got this value from running kubectl with bad args
// error: invalid label value: "test=-asdf": a valid label must be an empty string
// or consist of alphanumeric characters, '-', '_' or '.', and must start and end with
// an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is
// '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?'

var retagTemplate = template.Must(template.New("retagTemplate").Parse(
	`
<filter {{.Pattern}}>
  @type record_transformer
  enable_ruby true
  <record>
    kubernetes_pod_label_values {{range $i, $e := .Labels -}}${record.dig('kubernetes','labels','{{$e}}')&.gsub(/[.-]/, '_') || '_'}{{if isLast $i $.Labels }}{{else}}.{{end}}{{- end}}
  </record>
</filter>

<match {{.Pattern}}>
  @type rewrite_tag_filter
  <rule>
    key      kubernetes_pod_label_values
    pattern  ^(.+)$
    tag     ${tag}._labels.$1
  </rule>
</match>

<filter {{.Pattern}}.**>
  @type record_transformer
  remove_keys kubernetes_pod_label_values
</filter>
`))

func makeTagFromFilter(ns string, sortedLabelNames []string, labelNames map[string]string) string {
	buf := &bytes.Buffer{}

	if cont, ok := labelNames[util.ContainerLabel]; ok {
		// if the special label _container is used then its name goes to the
		// part of the tag that denotes the container
		buf.WriteString(fmt.Sprintf("kube.%s.*.%s._labels.", ns, cont))
	} else {
		buf.WriteString(fmt.Sprintf("kube.%s.*.*._labels.", ns))
	}

	for i, lb := range sortedLabelNames {
		if lb == util.ContainerLabel {
			continue
		}

		val, ok := labelNames[lb]
		if ok {
			buf.WriteString(safeLabelValue(val))
		} else {
			buf.WriteString("*")
		}

		if i < len(sortedLabelNames)-1 {
			buf.WriteString(".")
		}
	}

	return buf.String()
}

// replaces the empty string and all . with _
// as they have special meaning to fluentd
func safeLabelValue(s string) string {
	return reSafe.ReplaceAllString(s, "_")
}

func (p *expandLabelsMacroState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	allReferencedLabels := map[string]string{}
	collectLabels := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "filter" && d.Name != "match" {
			return nil
		}

		if !strings.HasPrefix(d.Tag, util.MacroLabels) {
			return nil
		}

		labelNames, err := util.ParseTagToLabels(d.Tag)
		if err != nil {
			return err
		}

		for lb := range labelNames {
			allReferencedLabels[lb] = ""
		}

		return nil
	}
	e := applyRecursivelyInPlace(input, p.Context, collectLabels)
	if e != nil {
		return nil, e
	}
	if len(allReferencedLabels) == 0 {
		return input, nil
	}

	delete(allReferencedLabels, util.ContainerLabel)
	sortedLabelNames := util.SortedKeys(allReferencedLabels)

	replaceLabels := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "filter" && d.Name != "match" {
			return nil
		}

		if !strings.HasPrefix(d.Tag, util.MacroLabels) {
			return nil
		}

		labelNames, err := util.ParseTagToLabels(d.Tag)
		if err != nil {
			// should never happen as the error should be caught beforehand
			return nil
		}

		d.Tag = makeTagFromFilter(ctx.Namespace, sortedLabelNames, labelNames)
		ctx.GenerationContext.augmentTag(d)
		return nil
	}
	applyRecursivelyInPlace(input, p.Context, replaceLabels)

	// prepare extra directives
	model := struct {
		Pattern string
		Labels  []string
	}{
		fmt.Sprintf("kube.%s.*.*", p.Context.Namespace),
		sortedLabelNames,
	}
	writer := &bytes.Buffer{}
	retagTemplate.Execute(writer, model)

	extraDirectives, err := fluentd.ParseString(writer.String())
	if err != nil {
		return nil, err
	}

	extraDirectives = append(extraDirectives, input...)

	return extraDirectives, nil
}
