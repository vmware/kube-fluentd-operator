package processors

import (
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

const (
	dirPlugin = "plugin"
)

// ExtractPlugins looks at the top-level directives in the admin namespace, deletes all <plugin>
// and stores the found plugin definitions under GenerationContext.Plugins map keyed by the plugin directive's path
func ExtractPlugins(g *GenerationContext, input fluentd.Fragment) fluentd.Fragment {
	plugins := map[string]*fluentd.Directive{}
	res := fluentd.Fragment{}

	//process only top-level plugin directives
	for _, dir := range input {
		if dir.Name == dirPlugin {
			plugins[dir.Tag] = dir
		} else {
			res = append(res, dir)
		}
	}

	g.Plugins = plugins

	return res
}

type expandPluginsState struct {
	BaseProcessorState
}

func (p *expandPluginsState) Process(input fluentd.Fragment) (fluentd.Fragment, error) {
	if len(p.Context.GenerationContext.Plugins) == 0 {
		// nothing to expand
		return input, nil
	}

	f := func(d *fluentd.Directive, ctx *ProcessorContext) error {
		if d.Name != "match" && d.Name != "store" {
			// only output plugins supported
			return nil
		}

		replacement, ok := p.Context.GenerationContext.Plugins[d.Type()]
		if !ok {
			return nil
		}

		// replace any nested content (buffers etc)
		// there is no option to redefine nested content
		d.Nested = replacement.Nested.Clone()

		// replace the params
		for k, v := range replacement.Params {
			// prefer the params defined at the call site
			if _, ok := d.Params[k]; !ok {
				d.Params[k] = v.Clone()
			}
		}

		// always change the type
		d.SetParam("@type", replacement.Type())

		// delete type parameter (someone is using the old syntax without @)
		delete(d.Params, "type")

		return nil
	}

	err := applyRecursivelyInPlace(input, p.Context, f)
	if err != nil {
		return nil, err
	}

	return input, nil
}
