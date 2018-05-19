// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"errors"
	"fmt"
	"strings"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

const (
	macroThisns = "$thisns"
)

type expandThisnsMacroState struct {
	BaseProcessorState
}

func (p *expandThisnsMacroState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	f := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		namespace := ctx.Namepsace

		if d.Name != "match" &&
			d.Name != "filter" {
			return nil
		}

		goodPrefix := fmt.Sprintf("kube.%s", namespace)

		if d.Tag == "**" || d.Tag == macroThisns {
			d.Tag = goodPrefix + ".**"
			ctx.GenerationContext.augmentTag(d)
			return nil
		}

		if strings.HasPrefix(d.Tag, macroThisns) {
			// handle the unusual case of $thisns.**
			d.Tag = goodPrefix + d.Tag[len(macroThisns):]
			ctx.GenerationContext.augmentTag(d)
			return nil
		}

		if strings.Index(d.Tag, "{") >= 0 {
			return errors.New("Cannot process {} in the tag yet")
		}

		if strings.HasPrefix(d.Tag, macroLabels) {
			// Let the labels processor handle this
			return nil
		}

		s := strings.Replace(d.Tag, macroThisns, goodPrefix, -1)

		if !strings.HasPrefix(s, goodPrefix+".") {
			return fmt.Errorf("bad tag for <%s>: %s. Tag must start with **, $thins or %s", d.Name, d.Tag, namespace)
		}

		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, f)
	if err != nil {
		return nil, err
	}

	return input, nil
}
