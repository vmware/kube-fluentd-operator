package processors

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

const (
	tagRegex = `(?:[^\s{}()]*(?:(?:(?:{.*?})|(?:\(.*?\)))[^\s{}()]*)+)|(?:[^\s{}()]+(?:(?:(?:{.*?})|(?:\(.*?\)))[^\s{}()]*)*)`
)

type expandTagsState struct {
	BaseProcessorState
	tagMatcher *regexp.Regexp
}

func (p *expandTagsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	if p.Context.AllowTagExpansion {
		return p.ProcessExpandingTags(input)
	} else {
		return p.ProcessNotExpandingTags(input)
	}
}

func (p *expandTagsState) ProcessExpandingTags(input fluentd.Fragment) (fluentd.Fragment, error) {
	f := func(d *fluentd.Directive, ctx *ProcessorContext) ([]*fluentd.Directive, error) {

		if d.Name != "match" && d.Name != "filter" {
			return []*fluentd.Directive{d}, nil
		}

		if p.tagMatcher == nil {
			p.tagMatcher = regexp.MustCompile(tagRegex)
		}

		expandingTags := p.tagMatcher.FindAllString(d.Tag, -1)
		remainders := p.tagMatcher.Split(d.Tag, -1)

		if len(strings.TrimSpace(strings.Join(remainders, ""))) > 0 {
			return nil, fmt.Errorf("Malformed tag %s. Cannot parse it", d.Tag)
		}

		var processingTags []string

		for len(expandingTags) > len(processingTags) {
			processingTags = expandingTags
			expandingTags = []string{}
			for _, t := range processingTags {
				expandedTags, err := expandFirstCurlyBraces(t)
				if err != nil {
					return nil, err
				}

				expandingTags = append(expandingTags, expandedTags...)
			}
		}

		if len(expandingTags) == 1 {
			return []*fluentd.Directive{d}, nil
		}

		expandedDirectives := make([]*fluentd.Directive, len(expandingTags))
		for i, t := range expandingTags {
			expandedDirectives[i] = d.Clone()
			expandedDirectives[i].Tag = t
		}

		return expandedDirectives, nil
	}

	output, err := applyRecursivelyWithState(input, p.Context, f)
	if err != nil {
		return nil, err
	}

	return output, nil
}

func expandFirstCurlyBraces(tag string) ([]string, error) {
	resultingTags := []string{}
	if open := strings.Index(tag, "{"); open >= 0 {
		if open > 0 && tag[open-1:open] == "#" {
			return nil, errors.New("Pattern #{...} is not yet supported in tag definition")
		}
		if close := strings.Index(tag, "}"); close > open+1 {
			expansionTerm := tag[open+1 : close]
			expansionTerms := strings.Split(expansionTerm, ",")

			for _, t := range expansionTerms {
				resultingTags = append(resultingTags, tag[:open]+strings.TrimSpace(t)+tag[close+1:])
			}
		} else {
			return nil, errors.New("Invalid {...} pattern in tag definition")
		}
	} else {
		resultingTags = append(resultingTags, tag)
	}

	return resultingTags, nil
}

func applyRecursivelyWithState(directives fluentd.Fragment, ctx *ProcessorContext, callback func(*fluentd.Directive, *ProcessorContext) ([]*fluentd.Directive, error)) (fluentd.Fragment, error) {
	if directives == nil {
		return nil, nil
	}

	for _, d := range directives {
		output, err := applyRecursivelyWithState(d.Nested, ctx, callback)
		if err != nil {
			return nil, err
		}
		d.Nested = output
	}

	newDirectives := []*fluentd.Directive{}
	for _, d := range directives {
		output, err := callback(d, ctx)
		if err != nil {
			return nil, err
		}
		newDirectives = append(newDirectives, output...)
	}

	return newDirectives, nil
}

func (p *expandTagsState) ProcessNotExpandingTags(input fluentd.Fragment) (fluentd.Fragment, error) {
	f := func(d *fluentd.Directive, ctx *ProcessorContext) error {

		if d.Name != "match" && d.Name != "filter" {
			return nil
		}

		if strings.Index(d.Tag, "{") >= 0 {
			return fmt.Errorf("Processing of {...} pattern in tags is disabled")
		}

		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, f)

	if err != nil {
		return nil, err
	}

	return input, nil
}
