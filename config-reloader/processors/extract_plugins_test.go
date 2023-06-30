package processors

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

func TestExtracPluginsFromKubeSystem(t *testing.T) {
	s := `
	<match kube.kube-system.**>
	  @type logzio_buffered
	</match>

	<plugin p1>
	  @type es
	</plugin>

	<plugin p2>
	  @type logzio
	</plugin>
	`
	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	g := &GenerationContext{}
	processed := ExtractPlugins(g, fragment)

	assert.Equal(t, 1, len(processed))

	assert.Equal(t, 2, len(g.Plugins))
	assert.Equal(t, "es", g.Plugins["p1"].Type())
	assert.Equal(t, "logzio", g.Plugins["p2"].Type())
}

func TestExpandPlugins(t *testing.T) {
	pluginDef := `
<plugin p1>
	@type es
	username admin
	buffer_path /hello/world
	buffer_size 1m
	<buffer>
		@type file
		path /var/log/fluentd.*.buffer
	</buffer>
</plugin>
`

	es, err := fluentd.ParseString(pluginDef)
	assert.Nil(t, err)

	// prepare context as if there is a plugin already defined in kube-system
	g := &GenerationContext{
		Plugins: map[string]*fluentd.Directive{
			"p1": es[0],
		},
	}

	nsConf := `
<filter **>
  @type grep
</filter>

<match **>
  # this is reference to the p1 plugin
  @type p1

  # param is copied
  param1 value1

  # param is overridden
  buffer_size 5m
</match>

<match **>
  @type some_type
</match>
	`

	fragment, err := fluentd.ParseString(nsConf)
	assert.Nil(t, err)

	ctx := &ProcessorContext{
		GenerationContext: g,
		Namespace:         "unit-test",
		DeploymentID:      "whatever",
	}

	state := &expandPluginsState{}
	state.SetContext(ctx)

	processed, err := state.Process(fragment)
	assert.Nil(t, err)

	// fmt.Printf("Processed:\n%s\n", processed)
	matchDir := processed[1]
	assert.Equal(t, "es", matchDir.Type())

	// param that's not overridden
	assert.Equal(t, "/hello/world", matchDir.Param("buffer_path"))

	// param that's defined at the call site
	assert.Equal(t, "value1", matchDir.Param("param1"))

	// param that's overridden
	assert.Equal(t, "5m", matchDir.Param("buffer_size"))

	// nested content is present
	assert.Equal(t, "buffer", matchDir.Nested[0].Name)
	assert.Equal(t, "/var/log/fluentd.*.buffer", matchDir.Nested[0].Param("path"))

	// types not found in the generation context are not touched
	matchDir = processed[2]
	assert.Equal(t, "some_type", matchDir.Type())
}

func TestNothingToExpand(t *testing.T) {
	// no plugins in this context
	g := &GenerationContext{}

	nsConf := `
<filter **>
  @type grep
</filter>

<match **>
  # this is reference to the p1 plugin
  @type p1

  # param is copied
  param1 value1

  # param is overridden
  buffer_size 5m
</match>

<match **>
  @type some_type
</match>
	`

	fragment, err := fluentd.ParseString(nsConf)
	assert.Nil(t, err)

	ctx := &ProcessorContext{
		GenerationContext: g,
		Namespace:         "unit-test",
		DeploymentID:      "whatever",
	}

	state := &expandPluginsState{}
	state.SetContext(ctx)

	processed, err := state.Process(fragment)
	assert.Nil(t, err)

	matchDir := processed[1]
	assert.Equal(t, "p1", matchDir.Type())

	// types not found in the generation context are not touched
	matchDir = processed[2]
	assert.Equal(t, "some_type", matchDir.Type())
}

func TestExpandPluginsInCopyStore(t *testing.T) {
	pluginDef := `
<plugin p1>
	@type es
	username admin
	buffer_path /hello/world
	buffer_size 1m
	<buffer>
		@type file
		path /var/log/fluentd.*.buffer
	</buffer>
</plugin>
<plugin p2>
	@type logzio
</plugin>
`

	es, err := fluentd.ParseString(pluginDef)
	assert.Nil(t, err)

	// prepare context as if there is a plugin already defined in kube-system
	g := &GenerationContext{
		Plugins: map[string]*fluentd.Directive{
			"p1": es[0],
			"p2": es[1],
		},
	}

	nsConf := `
<filter **>
  @type grep
</filter>

<match **>
  @type copy

  <store>
    # this is reference to the p1 plugin
    @type p1

    # param is copied
    param1 value1

    # param is overridden
    buffer_size 5m
  </store>
  <store>
    @type p2
  </store>
  <store>
    @type es
  </store>
</match>

<match **>
  @type some_type
</match>
	`

	fragment, err := fluentd.ParseString(nsConf)
	assert.Nil(t, err)

	ctx := &ProcessorContext{
		GenerationContext: g,
		Namespace:         "unit-test",
		DeploymentID:      "whatever",
	}

	state := &expandPluginsState{}
	state.SetContext(ctx)

	processed, err := state.Process(fragment)
	assert.Nil(t, err)

	// fmt.Printf("Processed:\n%s\n", processed)
	matchDir := processed[1]
	assert.Equal(t, "copy", matchDir.Type())

	// check first store component
	matchStore := matchDir.Nested[0]
	assert.Equal(t, "es", matchStore.Type())

	// param that's not overridden
	assert.Equal(t, "/hello/world", matchStore.Param("buffer_path"))

	// param that's defined at the call site
	assert.Equal(t, "value1", matchStore.Param("param1"))

	// param that's overridden
	assert.Equal(t, "5m", matchStore.Param("buffer_size"))

	// nested content is present
	assert.Equal(t, "buffer", matchStore.Nested[0].Name)
	assert.Equal(t, "/var/log/fluentd.*.buffer", matchStore.Nested[0].Param("path"))

	// check second store
	matchStore = matchDir.Nested[1]
	assert.Equal(t, "logzio", matchStore.Type())

	// types not found in the generation context are not touched
	matchDir = processed[2]
	assert.Equal(t, "some_type", matchDir.Type())
}
