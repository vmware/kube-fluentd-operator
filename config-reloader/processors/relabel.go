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

func normalizeLabelName(ctx *ProcessorContext, label string) string {
	if strings.HasPrefix(label, "@$") {
		// cross dependency to the share.go processor
		return label
	}

	return fmt.Sprintf("@%s-%s",
		util.MakeFluentdSafeName(label),
		util.Hash(ctx.Namepsace, label))
}

func (p *rewriteLabelsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	normalizeAllLabels := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "match" && d.Name != "store" {
			return nil
		}

		if d.Type() != "relabel" {
			return nil
		}

		labelName := d.Param("@label")
		if !strings.HasPrefix(labelName, "@") {
			return fmt.Errorf("bad label name: %s", labelName)
		}

		d.SetParam("@label", normalizeLabelName(ctx, labelName))
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
