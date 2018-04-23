// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"errors"

	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

type ProcessorContext struct {
	Namepsace      string
	AllowFile      bool
	DeploymentID   string
	MiniContainers []*datasource.MiniContainer
	KubeletRoot    string
}

type BaseProcessorState struct {
	Context *ProcessorContext
}

type FragmentProcessor interface {
	// SetContext is called once before processing begins
	SetContext(*ProcessorContext)

	// Prepare may define directives that are applied to the main fluentd file
	Prepare(fluentd.Fragment) (fluentd.Fragment, error)

	// Process defines directives that are put in their own ns-{namespace}.conf file
	Process(fluentd.Fragment) (fluentd.Fragment, error)
}

func (p *BaseProcessorState) Prepare(directives fluentd.Fragment) (fluentd.Fragment, error) {
	return directives, nil
}

func (p *BaseProcessorState) SetContext(ctx *ProcessorContext) {
	p.Context = ctx
}

func applyRecursivelyInPlace(directives fluentd.Fragment, ctx *ProcessorContext, callback func(*fluentd.Directive, *ProcessorContext) error) error {
	for _, d := range directives {
		err := callback(d, ctx)
		if err != nil {
			return err
		}
	}

	for _, d := range directives {
		err := applyRecursivelyInPlace(d.Nested, ctx, callback)
		if err != nil {
			return err
		}
	}

	return nil
}

func Process(input fluentd.Fragment, ctx *ProcessorContext, processors ...FragmentProcessor) (fluentd.Fragment, error) {
	if ctx == nil {
		return nil, errors.New("cannot work with nil ProcessorContext")
	}

	res := input
	var err error

	for _, proc := range processors {
		proc.SetContext(ctx)
		res, err = proc.Process(res)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func Prepare(input fluentd.Fragment, ctx *ProcessorContext, processors ...FragmentProcessor) (fluentd.Fragment, error) {
	if ctx == nil {
		return nil, errors.New("cannot work with nil ProcessorContext")
	}

	res := input
	var err error

	for _, proc := range processors {
		proc.SetContext(ctx)
		res, err = proc.Prepare(res)
		if err != nil {
			return nil, err
		}
	}

	return res, nil
}

func DefaultProcessors() []FragmentProcessor {
	return []FragmentProcessor{
		&expandThisnsMacroState{},
		&fixDestinations{},
		&expandLabelsMacroState{},
		&rewriteLabelsState{},
		&mountedFileState{},
	}
}
