// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func TestThisnsExpandOk(t *testing.T) {
	var s = `
	<match kube.monitoring.**>
	  @type null
	</match>

	<filter kube.monitoring.**>
	  @type null
	</filter>

	<match **>
	  @type null
	</match>
	
	<filter **>
		@type record_transformer
		enable_ruby true
		<record>
			hostname ${record["kubernetes"]["namespace_name"]}-${record["kubernetes"]["pod_name"]}
			program ${record["kubernetes"]["container_name"]}
			severity info
			facility local0
			message ${record['log']}
		</record>
	</filter>

	<filter $thisns.**>
		@type null
	</filter>

	<match $thisns.**>
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
	fragment, err = Process(fragment, ctx, &expandThisnsMacroState{})
	assert.Nil(t, err)
	fmt.Printf("Processed:\n%s", fragment)

	lit := fragment[0]
	assert.Equal(t, "kube.monitoring.**", lit.Tag)

	starstar := fragment[1]
	assert.Equal(t, "kube.monitoring.**", starstar.Tag)

	prefix := fragment[2]
	assert.Equal(t, "kube.monitoring.**", prefix.Tag)

	assert.True(t, strings.Index(fragment.String(), "$thisns") < 0)
}

func TestThisnsExpandBadConfig(t *testing.T) {

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
	}

	list := []string{
		`<match kubs-system>
          @type null
		 </match>`,
		`<match test.$thisns>
		   @type null
	     </match>`,
		`<match>
	       @type null
		 </match>`,
		`<match a.{b,c}>
	       @type null
	     </match>`,
	}

	for _, s := range list {
		fragment, err := fluentd.ParseString(s)
		assert.Nil(t, err)

		_, err = Process(fragment, ctx, &expandThisnsMacroState{})
		assert.NotNil(t, err)
	}
}
