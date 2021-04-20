// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

func TestMakeRewriteTagFragment(t *testing.T) {
	frag, err := makeRewriteTagFragment("src", "dest")
	assert.Nil(t, err)

	str := `<match kube.src.**>
  @type rewrite_tag_filter

  <rule>
    invert true
    key _dummy_
    pattern /ZZ/
    tag kube.dest.${tag_parts[2]}.${tag_parts[3]}
  </rule>
</match>

`
	assert.Equal(t, str, frag.String())
}

func TestExtractSourceNsFromMacro(t *testing.T) {
	data := []struct {
		Expr   string
		Result string
	}{
		{"asfag", ""},
		{"$from(a)", ""},
		{"@$from ( a  ", ""},
		{"@$from ()", ""},
		{"@$from ( )", ""},
		{"@$from a)", ""},
		{"@$from (a", ""},
		{"@$from(a)", "a"},
		{"@$from ( a ) ", "a"},
	}
	for _, m := range data {
		assert.Equal(t, m.Result, extractSourceNsFromMacro(m.Expr), "%s should parse into %s", m.Expr, m.Result)
	}
}

func TestMakeBridgeName(t *testing.T) {
	from := "from"
	to := "to"
	assert.Equal(t, "@bridge-from__to", makeBridgeName(from, to))
}

func TestProcessShareDirectiveFromReceivingNs(t *testing.T) {
	// sourceNsConf := `
	// <match $labels(msg=stdout)>
	//   @type copy
	//   <store>
	//     @type share
	//     with_namespace kfo-consumer
	//   </store>
	//   <store>
	//     @type share
	//     with_namespace no-such-namespace
	//   </store>
	// </match>
	// <match **>
	//   @type loggly
	//   loggly_url https://logs-01.loggly.com/inputs/$LOGGLY_TOKEN/tag/fluentd
	// </match>
	//`

	destNsConf := `
<label @$from(source-ns)>
  <match **>
    @type elasticsearch
    num_threads 8
  </match>
</label>
`

	gen := &GenerationContext{
		ReferencedBridges: map[string]bool{"@bridge-source-ns__dest-ns": true},
	}

	ctx := &ProcessorContext{
		Namespace:         "dest-ns",
		GenerationContext: gen,
	}

	input, err := fluentd.ParseString(destNsConf)
	assert.Nil(t, err)

	state := &shareLogsState{}
	state.SetContext(ctx)

	vt := state.GetValidationTrailer(input)
	assert.True(t, len(vt) == 0)

	processed, err := state.Process(input)
	assert.Nil(t, err)

	assert.Equal(t, "label", processed[0].Name)
	assert.Equal(t, "@bridge-source-ns__dest-ns", processed[0].Tag)

	assert.Equal(t, "match", processed[0].Nested[0].Name)
	assert.Equal(t, "kube.source-ns.**", processed[0].Nested[0].Tag)

	assert.Equal(t, "match", processed[0].Nested[1].Name)
	assert.Equal(t, "**", processed[0].Nested[1].Tag)
}

func TestProcessShareDirectiveFromPublishigNs(t *testing.T) {
	sourceNsConf := `
	<match $labels(msg=stdout)>
	  @type copy
	  <store>
	    @type share
	    with_namespace dest-ns
	  </store>
	  <store>
	    @type share
	    with_namespace no-such-namespace
	  </store>
	</match>
	`

	gen := &GenerationContext{
		ReferencedBridges: map[string]bool{"@bridge-source-ns__dest-ns": true},
	}

	ctx := &ProcessorContext{
		Namespace:         "source-ns",
		GenerationContext: gen,
	}

	input, err := fluentd.ParseString(sourceNsConf)
	assert.Nil(t, err)

	state := &shareLogsState{}
	state.SetContext(ctx)

	vt := state.GetValidationTrailer(input)
	assert.True(t, len(vt) == 1)
	assert.Equal(t, "label", vt[0].Name)
	assert.Equal(t, "@bridge-source-ns__dest-ns", vt[0].Tag)
	assert.Equal(t, "match", vt[0].Nested[0].Name)

	processed, err := state.Process(input)
	assert.Nil(t, err)

	assert.Equal(t, "copy", processed[0].Type())

	assert.Equal(t, 1, len(processed[0].Nested))
	assert.Equal(t, "relabel", processed[0].Nested[0].Type())
	assert.Equal(t, "@bridge-source-ns__dest-ns", processed[0].Nested[0].Param("@label"))
}

func TestProcessShareDirectiveCollectBridges(t *testing.T) {
	destNsConf := `<label @$from(source-ns)>
	<match **>
	  @type elasticsearch
	  num_threads 8
	</match>
  </label>
  `

	gen := &GenerationContext{
		ReferencedBridges: map[string]bool{},
	}

	ctx := &ProcessorContext{
		Namespace:         "dest-ns",
		GenerationContext: gen,
	}

	input, err := fluentd.ParseString(destNsConf)
	assert.Nil(t, err)

	state := &shareLogsState{}
	state.SetContext(ctx)

	_, err = state.Prepare(input)
	assert.Nil(t, err)

	assert.Equal(t, 1, len(gen.ReferencedBridges))
	assert.Equal(t, true, gen.ReferencedBridges["@bridge-source-ns__dest-ns"])
}
