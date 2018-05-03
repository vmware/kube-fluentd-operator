// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"errors"

	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

type GenerationContext struct {
	ReferencedBridges map[string]bool
}

type ProcessorContext struct {
	Namepsace         string
	NamespaceLabels   map[string]string
	AllowFile         bool
	DeploymentID      string
	MiniContainers    []*datasource.MiniContainer
	KubeletRoot       string
	GenerationContext *GenerationContext
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

	// GetValidationTrailer produces a trailer that would make a namspace config valid in isolation
	GetValidationTrailer(fluentd.Fragment) fluentd.Fragment
}

func (p *BaseProcessorState) Prepare(directives fluentd.Fragment) (fluentd.Fragment, error) {
	return nil, nil
}

func (p *BaseProcessorState) GetValidationTrailer(directives fluentd.Fragment) fluentd.Fragment {
	return nil
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

// Process chains the the processors outputs
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

// GetValidationTrailer accumulates the result from the processors
func GetValidationTrailer(input fluentd.Fragment, ctx *ProcessorContext, processors ...FragmentProcessor) fluentd.Fragment {
	if ctx == nil {
		return nil
	}

	res := fluentd.Fragment{}

	for _, proc := range processors {
		proc.SetContext(ctx)
		res = append(res, proc.GetValidationTrailer(input)...)
	}

	return res
}

// Prepare accumulates the result from the processors
func Prepare(input fluentd.Fragment, ctx *ProcessorContext, processors ...FragmentProcessor) (fluentd.Fragment, error) {
	if ctx == nil {
		return nil, errors.New("cannot work with nil ProcessorContext")
	}

	res := fluentd.Fragment{}

	for _, proc := range processors {
		proc.SetContext(ctx)
		prepDirectives, err := proc.Prepare(input)
		res = append(res, prepDirectives...)
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
		&shareLogsState{},
	}
}
