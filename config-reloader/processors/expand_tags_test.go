package processors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func TestTagsExpandOk(t *testing.T) {
	var input = []string{
		`<match kube.monitoring.{app1,app2}.** kube.monitoring.app3.**>
			@type null
		</match>`,
		`<filter kube.monitoring.{app1, app2}.**  kube.monitoring.app3.**>
	  		@type null
		</filter>`,
	}

	for _, s := range input {
		fragment, err := fluentd.ParseString(s)
		assert.Nil(t, err)

		fmt.Printf("Original:\n%s", fragment)

		ctx := &ProcessorContext{
			Namepsace:         "monitoring",
			GenerationContext: &GenerationContext{},
			AllowTagExpansion: true,
		}
		fragment, err = Process(fragment, ctx, &expandTagsState{})
		assert.Nil(t, err)
		assert.Equal(t, len(fragment), 3)
		fmt.Printf("Processed:\n%s", fragment)

		app1 := fragment[0]
		assert.Equal(t, "kube.monitoring.app1.**", app1.Tag)

		app2 := fragment[1]
		assert.Equal(t, "kube.monitoring.app2.**", app2.Tag)

		app3 := fragment[2]
		assert.Equal(t, "kube.monitoring.app3.**", app3.Tag)

		assert.True(t, strings.Index(fragment.String(), "{") < 0)
		assert.True(t, strings.Index(fragment.String(), "}") < 0)
	}
}

func TestNestedTagsExpandOk(t *testing.T) {
	var s = `
	<match **>
		@type relabel
		@label @test
	</match>

	<label @test>
	  <match kube.monitoring.{app1, app2}.** kube.monitoring.app3.**>
		@type null
	  </match>
	</label>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s", fragment)

	ctx := &ProcessorContext{
		Namepsace:         "monitoring",
		GenerationContext: &GenerationContext{},
		AllowTagExpansion: true,
	}
	fragment, err = Process(fragment, ctx, DefaultProcessors()...)
	assert.Nil(t, err)
	assert.Equal(t, len(fragment), 2)
	assert.Equal(t, len(fragment[1].Nested), 3)
	fmt.Printf("Processed:\n%s", fragment)

	app1 := fragment[1].Nested[0]
	assert.Equal(t, "kube.monitoring.app1.**", app1.Tag)

	app2 := fragment[1].Nested[1]
	assert.Equal(t, "kube.monitoring.app2.**", app2.Tag)

	app3 := fragment[1].Nested[2]
	assert.Equal(t, "kube.monitoring.app3.**", app3.Tag)

	assert.True(t, strings.Index(fragment.String(), "{") < 0)
	assert.True(t, strings.Index(fragment.String(), "}") < 0)
}

func TestTagsExpandBadConfig(t *testing.T) {

	ctx := &ProcessorContext{
		Namepsace:         "monitoring",
		AllowTagExpansion: true,
	}

	list := []string{
		`<match kube.monitoring.#{ENV_VAR}.**>
          @type null
		 </match>`,
		`<match kube.monitoring.{app1.**>
		  @type null
		</match>`,
		`<match kube.monitoring.app2}.**>
		  @type null
		</match>`,
	}

	for _, s := range list {
		fragment, err := fluentd.ParseString(s)
		assert.Nil(t, err)

		_, err = Process(fragment, ctx, &expandTagsState{})
		assert.NotNil(t, err)
	}
}
