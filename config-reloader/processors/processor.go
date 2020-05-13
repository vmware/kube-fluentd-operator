// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"errors"
	"fmt"

	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

const (
	prefixProcessed = "_proc"
)

// GenerationContext holds state for one loop of the controller.
// It is shared accross all processors and all encountered namespaces.
// Different processorts can share state using this class.
// Use this class sparingly.
type GenerationContext struct {
	ReferencedBridges map[string]bool
	NeedsProcessing   bool
	Plugins           map[string]*fluentd.Directive
}

func (g *GenerationContext) augmentTag(d *fluentd.Directive) {
	if g == nil || !g.NeedsProcessing {
		return
	}

	orig := d.Tag
	d.Tag = augmentTag(orig)
}

// ProcessorContext is how a processor gets an environemnt to operate in.
// It is both the model and the workspace of a processor.
type ProcessorContext struct {
	Namepsace         string
	NamespaceLabels   map[string]string
	AllowFile         bool
	DeploymentID      string
	MiniContainers    []*datasource.MiniContainer
	KubeletRoot       string
	GenerationContext *GenerationContext
	AllowTagExpansion bool
}

type BaseProcessorState struct {
	Context *ProcessorContext
}

// FragmentProcessor converts an instance of a Fluentd configuration
// into a different one. This is how rewriting of user-provided configuration happens.
type FragmentProcessor interface {
	// SetContext is called once before processing begins
	SetContext(*ProcessorContext)

	// Prepare may define directives that are applied to the main fluentd file
	// The results of all registered processors are concatenated ans included in fluentd.conf
	Prepare(fluentd.Fragment) (fluentd.Fragment, error)

	// Process defines directives that are put in their own ns-{namespace}.conf file
	// The results of the registered processors are chained. Order of registration is important.
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

func transform(input fluentd.Fragment, f func(dir *fluentd.Directive, parent *fluentd.Fragment) *fluentd.Directive) fluentd.Fragment {
	res := &fluentd.Fragment{}
	doTransform(input, f, res)
	return *res
}

func doTransform(input fluentd.Fragment, f func(dir *fluentd.Directive, parent *fluentd.Fragment) *fluentd.Directive, res *fluentd.Fragment) {
	for _, child := range input {
		newChild := f(child, res)

		if len(child.Nested) > 0 && newChild != nil {
			chres := &fluentd.Fragment{}
			doTransform(child.Nested, f, chres)
			newChild.Nested = *chres
		}
	}
}

func copy(dir *fluentd.Directive, parent *fluentd.Fragment) *fluentd.Directive {
	clone := dir.Clone()
	*parent = append(*parent, clone)
	return clone
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

func augmentTag(orig string) string {
	if orig == "" {
		// that's an error usually
		return ""
	}

	return fmt.Sprintf("%s %s.%s", orig, prefixProcessed, orig)
}

// DefaultProcessors return all currently known processors. You can compise a list
// of processors but be aware of dependencies between processors (order matters).
func DefaultProcessors() []FragmentProcessor {
	return []FragmentProcessor{
		&expandPluginsState{},
		&expandTagsState{},
		&expandThisnsMacroState{},
		&fixDestinations{},
		&expandLabelsMacroState{},
		&uniqueRewriteTagState{},
		&rewriteLabelsState{},
		&mountedFileState{},
		&shareLogsState{},
		&detectExceptionsState{},
	}
}
