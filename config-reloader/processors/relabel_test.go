// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func TestParseConfigWithBadLabels(t *testing.T) {
	input := []string{
		`
		<label no-at-prefix>
		</label>
		`,
		`
		<match **>
		 @type relabel
		 # missing label name
		</match>
		`,
		`
		<match **>
		 @type relabel
		 @label hello
		 # bad prefix
		</match>
		`,
	}

	ctx := &ProcessorContext{
		Namepsace: "demo",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}
	for _, s := range input {
		fragment, err := fluentd.ParseString(s)
		assert.Nil(t, err, "must parse, failed instead with %+v", err)

		fragment, err = Process(fragment, ctx, &rewriteLabelsState{})
		assert.Nil(t, fragment)
		assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
	}
}

func TestParseConfigWithGoodLabels(t *testing.T) {
	s := `
		<match **>
		 @type relabel
		 @label @hello
		</match>
		`

	ctx := &ProcessorContext{
		Namepsace: "demo",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err, "must parse, failed instead with %+v", err)

	fragment, err = Process(fragment, ctx, &rewriteLabelsState{})
	assert.Nil(t, err, "must succeed, got error instead: %+v", err)
	assert.NotNil(t, fragment)
}

func TestLabelsAreRewritten(t *testing.T) {
	var s = `
	<match kube.monitoring.*.prometheus>
	  @type relabel
	  @label @prometheus
	</match>

	<label @prometheus>
	  <match **>
		@type forward
		# forward prometheus logs somewhere
	  </match>
	</label>

	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}
	fragment, err = Process(fragment, ctx, &rewriteLabelsState{})
	assert.Nil(t, err)
	fmt.Printf("Processed:\n%s\n", fragment)

	lit := fragment[0]
	assert.Equal(t, "kube.monitoring.*.prometheus", lit.Tag)
	assert.NotEqual(t, lit.Param("@label"), "@prometheus")

	starstar := fragment[1]
	assert.Equal(t, "label", starstar.Name)
	assert.NotEqual(t, lit.Tag, "@prometheus")

	match := fragment[1].Nested[0]
	assert.Equal(t, "**", match.Tag)
}

func TestCopyPluginLabelsAreRewritten(t *testing.T) {
	var s = `
	<match kube.monitoring.**>
	  @type copy
	  <store>
		@type relabel
		@label @output
	  </store>
	  <store>
		@type relabel
		@label @postprocessing
	  </store>
	</match>

	<label @output>
	  <match **>
		@type forward
		# forward logs to output
	  </match>
	</label>

	<label @postprocessing>
	  <match **>
		@type null
		# perform additional processing for other output
	  </match>
	</label>

	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}
	fragment, err = Process(fragment, ctx, &rewriteLabelsState{})
	assert.Nil(t, err)
	fmt.Printf("Processed:\n%s\n", fragment)

	outputRelabel := fragment[0].Nested[0].Param("@label")
	postprocessingRelabel := fragment[0].Nested[1].Param("@label")

	outputLabel := fragment[1].Tag
	postprocessingLabel := fragment[2].Tag

	assert.Equal(t, outputRelabel, outputLabel)
	assert.Equal(t, postprocessingRelabel, postprocessingLabel)
	assert.NotEqual(t, outputRelabel, "@output")
	assert.NotEqual(t, postprocessingRelabel, "@postprocessing")
}

func TestLabelWithLabelsAndRelabelsAndElse(t *testing.T) {
	s := `
<match $labels(app=grafana, release=rel, _container=main)>
	@type relabel
	@label @mon
</match>

<match $labels(app=prom, heritage=Helm)>
	@type relabel
	@label @mon
</match>

<filter kube.demo.**>
  @type null
</filter>

<label @mon>
	<match **>
	  @type logzio
  </match>
</label>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "demo",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	fragment, err = Process(fragment, ctx, DefaultProcessors()...)
	assert.Nil(t, err)

	fmt.Printf("Processed:\n%s\n", fragment)
}

func TestNastyRegex(t *testing.T) {
	s := `
<match $labels(app=helm)>
  @type relabel
  @label @test
</match>

<label @test>
  <filter **>
    @parser
    format /^(?<host>[^ ]*) [^ ]* (?<user>[^ ]*) \[(?<time>[^\]]*)\] "(?<method>\S+)(?: +(?<path>[^ ]*) +\S*)?" (?<code>[^ ]*) (?<size>[^ ]*)$/
    time_format %d/%b/%Y:%H:%M:%S %z
    key_name message
  </filter>
</label>
`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s\n", fragment)

	ctx := &ProcessorContext{
		Namepsace: "demo",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	_, err = Process(fragment, ctx, DefaultProcessors()...)
	assert.Nil(t, err)
}
