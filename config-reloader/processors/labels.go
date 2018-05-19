// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"text/template"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

const (
	macroLabels    = "$labels"
	containerLabel = "_container"
)

type expandLabelsMacroState struct {
	BaseProcessorState
}

var reSafe = regexp.MustCompile("[.-]|^$")

// got this value from running kubectl with bad args
// error: invalid label value: "test=-asdf": a valid label must be an empty string
// or consist of alphanumeric characters, '-', '_' or '.', and must start and end with
// an alphanumeric character (e.g. 'MyValue',  or 'my_value',  or '12345', regex used for validation is
// '(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?'

var reValidLabelName = regexp.MustCompile("^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$")
var reValidLabelValue = regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")

var fns = template.FuncMap{
	"last": func(x int, a interface{}) bool {
		return x == reflect.ValueOf(a).Len()-1
	},
}

var retagTemplate = template.Must(template.New("retagTemplate").Funcs(fns).Parse(
	`
<filter {{.Pattern}}>
  @type record_transformer
  enable_ruby true
  <record>
    kubernetes_pod_label_values {{range $i, $e := .Labels -}}${record["kubernetes"]["labels"]["{{$e}}"]&.gsub(/[.-]/, '_') || '_'}{{if last $i $.Labels }}{{else}}.{{end}}{{- end}}
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

func parseTagToLabels(tag string) (map[string]string, error) {
	if !strings.HasPrefix(tag, macroLabels) {
		return nil, nil
	}

	if !strings.HasPrefix(tag, macroLabels+"(") &&
		!strings.HasSuffix(tag, ")") {
		return nil, fmt.Errorf("bad $labels macro use: %s", tag)
	}

	labelsOnly := tag[len(macroLabels)+1 : len(tag)-1]

	result := map[string]string{}

	records := strings.Split(labelsOnly, ",")
	for _, rec := range records {
		if rec == "" {
			// be generous
			continue
		}
		kv := strings.Split(rec, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("bad label definition: %s", kv)
		}

		k := util.Trim(kv[0])
		if k != containerLabel {
			if !reValidLabelName.MatchString(k) {
				return nil, fmt.Errorf("bad label name: %s", k)
			}
		}

		v := util.Trim(kv[1])
		if !reValidLabelValue.MatchString(v) {
			return nil, fmt.Errorf("bad label value: %s", v)
		}
		if k == containerLabel && v == "" {
			return nil, fmt.Errorf("value for %s cannot be empty string", containerLabel)
		}

		result[k] = v
	}

	if len(result) == 0 {
		return nil, errors.New("at least one label must be given")
	}

	return result, nil
}

func makeTagFromFilter(ns string, sortedLabelNames []string, labelNames map[string]string) string {
	buf := &bytes.Buffer{}

	if cont, ok := labelNames[containerLabel]; ok {
		// if the special label _container is used then its name goes to the
		// part of the tag that denotes the container
		buf.WriteString(fmt.Sprintf("kube.%s.*.%s._labels.", ns, cont))
	} else {
		buf.WriteString(fmt.Sprintf("kube.%s.*.*._labels.", ns))
	}

	for i, lb := range sortedLabelNames {
		if lb == containerLabel {
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

		if d.Tag == "" {
			return nil
		}

		labelNames, err := parseTagToLabels(d.Tag)
		if err != nil {
			return err
		}

		for lb := range labelNames {
			allReferencedLabels[lb] = ""
		}

		return nil
	}
	applyRecursivelyInPlace(input, p.Context, collectLabels)
	if len(allReferencedLabels) == 0 {
		return input, nil
	}

	delete(allReferencedLabels, containerLabel)
	sortedLabelNames := util.SortedKeys(allReferencedLabels)

	replaceLabels := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "filter" && d.Name != "match" {
			return nil
		}

		if d.Tag == "" {
			return nil
		}

		labelNames, err := parseTagToLabels(d.Tag)
		if err != nil {
			// nothing to replace, it was not a $labels macro
			return nil
		}

		d.Tag = makeTagFromFilter(ctx.Namepsace, sortedLabelNames, labelNames)
		ctx.GenerationContext.augmentTag(d)
		return nil
	}
	applyRecursivelyInPlace(input, p.Context, replaceLabels)

	// prepare extra directives
	model := struct {
		Pattern string
		Labels  []string
	}{
		fmt.Sprintf("kube.%s.*.*", p.Context.Namepsace),
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
