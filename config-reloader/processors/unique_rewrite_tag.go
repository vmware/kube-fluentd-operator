package processors

import (
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

const (
	macroUniqueTag = "$tag"
)

type uniqueRewriteTagState struct {
	BaseProcessorState
}

func (p *uniqueRewriteTagState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	adaptUniqueRewriteTagPlugin := func(d *fluentd.Directive, ctx *ProcessorContext) error {

		if d.Name != "match" || d.Type() != "retag" {
			return nil
		}

		for _, rule := range d.Nested {
			if rule.Name != "rule" {
				continue
			}

			tagParam := rule.Param("tag")
			if tagParam == "" {
				return fmt.Errorf("retag plugin requires each rule to have a tag parameter")
			}

			if strings.Index(tagParam, "${tag_parts[") >= 0 || strings.Index(tagParam, "__TAG_PARTS[") >= 0 {
				return fmt.Errorf("retag plugin does not yet support the ${tag_parts[n]} and __TAG_PARTS[n]__ placeholders")
			}

			targetTag := p.createUniqueTag(tagParam, ctx.Namepsace)

			rule.SetParam("tag", targetTag)
		}

		d.SetParam("@type", "rewrite_tag_filter")

		return nil
	}

	rewriteTagMacro := func(d *fluentd.Directive, ctx *ProcessorContext) error {

		if d.Name != "match" && d.Name != "filter" {
			return nil
		}

		if !strings.HasPrefix(d.Tag, macroUniqueTag) {
			return nil
		}

		if !strings.HasSuffix(d.Tag, ")") {
			return fmt.Errorf("Malformed tag. To match output from the retag plugin the tag must be placed inside the $tag() macro")
		}

		targetTag := d.Tag[len(macroUniqueTag)+1 : len(d.Tag)-1]

		d.Tag = p.createUniqueTag(targetTag, ctx.Namepsace)
		ctx.GenerationContext.augmentTag(d)

		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, adaptUniqueRewriteTagPlugin)
	if err != nil {
		return nil, err
	}

	err = applyRecursivelyInPlace(input, p.Context, rewriteTagMacro)
	if err != nil {
		return nil, err
	}

	return input, nil
}

func (p *uniqueRewriteTagState) createUniqueTag(tag, namespace string) string {
	return "kube." + namespace + "._retag." + tag
}
