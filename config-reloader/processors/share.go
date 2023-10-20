// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/template"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

const (
	macroFrom = "$from"
	// typeShare = "share"
)

var rewriteSharedTag = template.Must(template.New("name").Parse(`
	<match {{ .SourceTag }}>
		@type rewrite_tag_filter
		<rule>
			key _dummy_
			pattern /ZZ/
			invert  true
			tag kube.{{ .Namespace }}.${tag_parts[2]}.${tag_parts[3]}
		</rule>
	</match>
	`))

type bridge struct {
	SourceTag string
	Namespace string
}

type shareLogsState struct {
	BaseProcessorState
}

func makeBridgeName(sourceNs, destNs string) string {
	return fmt.Sprintf("@bridge-%s__%s", sourceNs, destNs)
}

func extractSourceNsFromMacro(labelExpr string) string {
	pfx := "@" + macroFrom
	if !strings.HasPrefix(labelExpr, pfx) {
		return ""
	}

	start := strings.LastIndexByte(labelExpr, '(')
	if start <= 0 {
		return ""
	}

	end := strings.LastIndexByte(labelExpr, ')')
	if end <= 0 {
		return ""
	}

	return util.Trim(labelExpr[start+1 : end])
}

func makeRewriteTagFragment(sourceNs string, destNs string) (fluentd.Fragment, error) {
	buf := &bytes.Buffer{}
	rewriteSharedTag.Execute(buf, &bridge{
		Namespace: destNs,
		SourceTag: fmt.Sprintf("kube.%s.**", sourceNs),
	})

	return fluentd.ParseString(buf.String())
}

func (p *shareLogsState) Prepare(input fluentd.Fragment) (fluentd.Fragment, error) {
	collectReferencedBridges := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "label" {
			return nil
		}

		sourceNs := extractSourceNsFromMacro(d.Tag)
		if sourceNs == "" {
			return nil
		}

		bridge := makeBridgeName(sourceNs, p.Context.Namespace)
		p.Context.GenerationContext.ReferencedBridges[bridge] = true
		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, collectReferencedBridges)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (p *shareLogsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	rewriteShareType := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "match" {
			return nil
		}

		if d.Type() != "copy" {
			return nil
		}

		newContent := fluentd.Fragment{}
		for _, nested := range d.Nested {
			if nested.Name != "store" {
				newContent = append(newContent, nested)
				continue
			}

			if nested.Type() != "share" {
				newContent = append(newContent, nested)
				continue
			}

			destNs := nested.Param("with_namespace")
			if destNs == "" {
				return fmt.Errorf("@type share required a with_namespace parameter")
			}
			bridge := makeBridgeName(p.Context.Namespace, destNs)

			// only retain @relabel stores when the bridges are being referenced
			if _, ok := p.Context.GenerationContext.ReferencedBridges[bridge]; ok {
				obj := &fluentd.Directive{
					Name:   "store",
					Params: fluentd.Params{},
				}
				obj.SetParam("@type", "relabel")
				obj.SetParam("@label", bridge)
				newContent = append(newContent, obj)
			}
		}

		d.Nested = newContent
		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, rewriteShareType)
	if err != nil {
		return nil, err
	}

	rewriteFromMacro := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "label" {
			return nil
		}

		sourceNs := extractSourceNsFromMacro(d.Tag)
		if sourceNs == "" {
			return nil
		}

		bridge := makeBridgeName(sourceNs, p.Context.Namespace)
		d.Tag = bridge

		fragment, err := makeRewriteTagFragment(sourceNs, p.Context.Namespace)
		if err != nil {
			// or just panic??
			return err
		}

		// prepend the tag-rewriter at the top
		d.Nested = append(fragment, d.Nested...)

		return nil
	}

	err = applyRecursivelyInPlace(input, p.Context, rewriteFromMacro)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (p *shareLogsState) GetValidationTrailer(directives fluentd.Fragment) fluentd.Fragment {
	res := fluentd.Fragment{}

	for k := range p.Context.GenerationContext.ReferencedBridges {
		if strings.HasPrefix(k, fmt.Sprintf("@bridge-%s__", p.Context.Namespace)) {
			dir := &fluentd.Directive{
				Name: "label",
				Tag:  k,
				Nested: fluentd.Fragment{
					&fluentd.Directive{
						Name:   "match",
						Tag:    "**",
						Params: fluentd.ParamsFromKV("@type", "null"),
					},
				},
			}

			res = append(res, dir)
		}
	}

	return res
}
