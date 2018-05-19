// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

const (
	keyDetExc = "detexc"
)

type detectExceptionsState struct {
	BaseProcessorState
}

func (state *detectExceptionsState) Prepare(input fluentd.Fragment) (fluentd.Fragment, error) {
	cb := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name == "filter" && d.Type() == "detect_exceptions" {
			ctx.GenerationContext.NeedsProcessing = true
		}

		return nil
	}

	applyRecursivelyInPlace(input, state.Context, cb)
	return nil, nil
}

func (state *detectExceptionsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	rewrite := func(dir *fluentd.Directive, parent *fluentd.Fragment) *fluentd.Directive {
		if dir.Name != "filter" || dir.Type() != "detect_exceptions" {
			c := dir.Clone()
			*parent = append(*parent, c)
			return c
		}

		unprocessedSelector := extractSelector(dir.Tag)
		tagPrefix := makeTagPrefix(unprocessedSelector)

		rule := &fluentd.Directive{
			Name:   "rule",
			Params: fluentd.Params{},
		}
		rule.SetParam("key", "_dummy")
		rule.SetParam("pattern", "/ZZ/")
		rule.SetParam("invert", "true")
		rule.SetParam("tag", fmt.Sprintf("%s.%s.${tag}", tagPrefix, prefixProcessed))

		rewriteTag := &fluentd.Directive{
			Name:   "match",
			Tag:    unprocessedSelector,
			Params: fluentd.ParamsFromKV("@type", "rewrite_tag_filter"),
			Nested: fluentd.Fragment{rule},
		}

		detectExceptions := &fluentd.Directive{
			Name:   "match",
			Tag:    fmt.Sprintf("%s.%s.%s", tagPrefix, prefixProcessed, unprocessedSelector),
			Params: fluentd.ParamsFromKV("@type", "detect_exceptions"),
		}
		detectExceptions.SetParam("stream", "container_info")
		detectExceptions.SetParam("remove_tag_prefix", tagPrefix)

		// copy all relevant params from the original <filter> directive
		// https://github.com/GoogleCloudPlatform/fluent-plugin-detect-exceptions
		copyParam("languages", dir, detectExceptions)
		copyParam("multiline_flush_interval", dir, detectExceptions)
		copyParam("max_lines", dir, detectExceptions)
		copyParam("max_bytes", dir, detectExceptions)
		copyParam("message", dir, detectExceptions)

		*parent = append(*parent, rewriteTag, detectExceptions)

		return nil
	}

	res := transform(input, rewrite)
	return res, nil
}

func makeTagPrefix(selector string) string {
	return util.Hash(keyDetExc, selector)
}

func extractSelector(tag string) string {
	parts := strings.Split(tag, " ")
	// abstraction leak: the labels processor has produced a tag in the form "xxx _proc.xxx"
	// the auto-generated <match> directives need only the first one
	return parts[0]
}

func copyParam(name string, src, dest *fluentd.Directive) {
	val := src.Param(name)
	if val != "" {
		dest.SetParam(name, val)
	}
}
