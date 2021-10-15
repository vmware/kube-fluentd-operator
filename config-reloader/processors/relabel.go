// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
)

type rewriteLabelsState struct {
	BaseProcessorState
}

var validLabelDirectives = []string{"match", "store", "filter", "parse", "source"}
var validLabelTypes = []string{"relabel", "null", "forward", "stdout", "copy", "kafka", "elasticsearch"}

func contains(slice []string, item string) bool {
	set := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		set[s] = struct{}{}
	}

	_, ok := set[item]
	return ok
}

func normalizeLabelName(ctx *ProcessorContext, label string) string {
	if strings.HasPrefix(label, "@$") {
		// cross dependency to the share.go processor
		return label
	}

	return fmt.Sprintf("@%s-%s",
		util.MakeFluentdSafeName(label),
		util.Hash(ctx.Namespace, label))
}

func (p *rewriteLabelsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	normalizeAllLabels := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if !contains(validLabelDirectives, d.Name) {
			return nil
		}

		// Process any timeout_labels since they are valid labels:
		timeoutLabel := d.Param("timeout_label")
		if timeoutLabel != "" {
			if !strings.HasPrefix(timeoutLabel, "@") {
				return fmt.Errorf("bad label name %s for timeout_label, must start with @", timeoutLabel)
			}

			d.SetParam("timeout_label", normalizeLabelName(ctx, timeoutLabel))
		}

		// Continue parsing normal @labels:
		if !contains(validLabelTypes, d.Type()) {
			return nil
		}

		labelName := d.Param("@label")
		if labelName != "" {
			if !strings.HasPrefix(labelName, "@") {
				return fmt.Errorf("bad label name %s for @label, must start with @", labelName)
			}

			d.SetParam("@label", normalizeLabelName(ctx, labelName))
		}

		return nil
	}

	rewriteLabelTag := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "label" {
			return nil
		}

		labelName := d.Tag
		if !strings.HasPrefix(labelName, "@") {
			return fmt.Errorf("bad label name %s for <label>, must start with @", labelName)
		}

		d.Tag = normalizeLabelName(ctx, labelName)
		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, normalizeAllLabels)
	if err != nil {
		return nil, err
	}

	err = applyRecursivelyInPlace(input, p.Context, rewriteLabelTag)
	if err != nil {
		return nil, err
	}

	return input, nil
}
