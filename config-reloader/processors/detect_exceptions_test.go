// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

func TestRewrite(t *testing.T) {
	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	s := `
<filter $labels(app=jpetstore)>
	@type detect_exceptions
	languages java, python
</filter>

<filter $labels(server=apache)>
	@type parse
	format apache2
</filter>

<match **>
  @type null
</match>
`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	detExc := &detectExceptionsState{}
	labelProc := &expandLabelsMacroState{}
	expandThis := &expandThisnsMacroState{}

	_, err = Prepare(fragment, ctx, expandThis, labelProc, detExc)
	assert.Nil(t, err)

	processed, err := Process(fragment, ctx, expandThis, labelProc, detExc)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(processed))

	fmt.Printf("Processed:\n%s\n", processed)
}

func TestWithoutExceptions(t *testing.T) {

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
		AllowTagExpansion: true,
	}

	s := `
	<filter $labels(server=apache)>
		@type parse
		format apache2
	</filter>

	<match **>
	  @type null
	</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	procs := DefaultProcessors()
	_, err = Prepare(fragment, ctx, procs...)
	assert.Nil(t, err)

	processed, err := Process(fragment, ctx, procs...)
	assert.Nil(t, err)
	assert.Equal(t, 5, len(processed))

	fmt.Printf("Processed:\n%s\n", processed)

	last := processed[len(processed)-1]
	assert.Equal(t, "kube.monitoring.**", last.Tag)
}

func TestWithExceptions(t *testing.T) {

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
		AllowTagExpansion: true,
	}

	s := `
	<filter $labels(server=apache)>
		@type parse
		format apache2
	</filter>

	<filter $labels(app=django)>
		@type detect_exceptions
		language python
	</filter>

	<match **>
	  @type null
	</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	procs := DefaultProcessors()
	_, err = Prepare(fragment, ctx, procs...)
	assert.Nil(t, err)

	processed, err := Process(fragment, ctx, procs...)
	assert.Nil(t, err)
	assert.Equal(t, 7, len(processed))

	fmt.Printf("Processed:\n%s\n", processed)

	last := processed[len(processed)-1]
	assert.Equal(t, "kube.monitoring.** _proc.kube.monitoring.**", last.Tag)

	detExc := processed[len(processed)-2]
	assert.Equal(t, "container_info", detExc.Param("stream"))
	assert.Equal(t, "detect_exceptions", detExc.Type())
}

func TestBuild(t *testing.T) {
	var s = `
<match **>
    @type logzio
	<buffer>
		@type file
		path /etc/passwd
		<nested>
		</nested>
  </buffer>
</match>

<match **>
  @type logzio
	<buffer>
		@type file
		path /etc/passwd
  </buffer>
</match>
`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	clone := transform(fragment, copy)

	assert.Equal(t, fragment.String(), clone.String())
}

func TestExtractSelector(t *testing.T) {
	assert.Equal(t, "xxx", extractSelector("xxx"))
	assert.Equal(t, "xxx", extractSelector("xxx _proc.xxx"))
	assert.Equal(t, "xxx", extractSelector("xxx what ever man"))
}
