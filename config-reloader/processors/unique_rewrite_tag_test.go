package processors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func TestTagsRewrittenOk(t *testing.T) {
	var s = `
	<match kube.monitoring.**>
      @type retag
      <rule>
        key message
        pattern ^ERROR
        tag notifications.error
      </rule>
      <rule>
        key message
        pattern ^FATAL
        tag notifications.fatal
      </rule>
    </match>

    <filter $tag(notifications.error)>
      @type null
    </filter>

    <filter $tag(notifications.fatal)>
	  @type null
    </filter>

    <match $tag(notifications.**)>
	  @type null
    </match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s", fragment)

	ctx := &ProcessorContext{
		Namepsace:         "monitoring",
		GenerationContext: &GenerationContext{},
	}
	fragment, err = Process(fragment, ctx, &uniqueRewriteTagState{})
	assert.Nil(t, err)
	fmt.Printf("Processed:\n%s", fragment)

	rewritePlugin := fragment[0]
	assert.Equal(t, "rewrite_tag_filter", rewritePlugin.Param("@type"))

	rule1 := rewritePlugin.Nested[0]
	rule2 := rewritePlugin.Nested[1]

	filter1 := fragment[1]
	filter2 := fragment[2]

	assert.Equal(t, rule1.Param("tag"), filter1.Tag)
	assert.Equal(t, rule2.Param("tag"), filter2.Tag)
	assert.True(t, strings.Index(filter1.Tag, macroUniqueTag) < 0)
	assert.True(t, strings.Index(filter2.Tag, macroUniqueTag) < 0)
	assert.NotEqual(t, strings.Split(filter1.Tag, ".")[0], "notifications")
	assert.NotEqual(t, strings.Split(filter2.Tag, ".")[0], "notifications")

	match := fragment[3]

	assert.Equal(t, strings.Split(filter1.Tag, ".error")[0], strings.Split(match.Tag, ".**")[0])
	assert.True(t, strings.Index(match.Tag, macroUniqueTag) < 0)
}

func TestRewriteTagsBadConfig(t *testing.T) {

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
		GenerationContext: &GenerationContext{
			ReferencedBridges: map[string]bool{},
		},
	}

	list := []string{
		`<match kube.monitoring.**>
		  @type rewrite_tag_filter
		  <rule>
		    key message
			pattern ^ERROR
			tag notifications.error
		  </rule>
		</match>`,
		`<match kube.monitoring.**>
		  @type retag
		  <rule>
		    key message
		    pattern ^ERROR
		  </rule>
		</match>`,
		`<match kube.monitoring.**>
		  @type retag
		  <rule>
		    key message
			pattern ^ERROR
			tag notifications.${tag_parts[1]}
		  </rule>
		</match>`,
	}

	for _, s := range list {
		fragment, err := fluentd.ParseString(s)
		assert.Nil(t, err)

		_, err = Process(fragment, ctx, DefaultProcessors()...)
		assert.NotNil(t, err)
	}
}
