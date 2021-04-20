// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func TestDestinationsRewriteBufferPath(t *testing.T) {
	var s = `
<match **>
  @type logzio
	<buffer>
		@type file
		path /etc/passwd
  </buffer>
</match>

<match **>
  @type kafka
  buffer_path /etc/hosts
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

<match **>
  @type whatever
</match>
`
	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s", fragment)

	ctx := &ProcessorContext{
		Namespace: "monitoring",
	}
	fragment, err = Process(fragment, ctx, &fixDestinations{})
	assert.Nil(t, err)
	fmt.Printf("Processed: %s", fragment)

	logzio := fragment[0]
	assert.NotEqual(t, "/etc/passwd", logzio.Nested[0].Param("path"))

	kafka := fragment[1]
	assert.NotEqual(t, "/etc/hosts", kafka.Param(paramBufferPath))

	whatever := fragment[2]
	assert.Equal(t, "", whatever.Param(paramBufferPath))
}

func TestExpandBadConfig(t *testing.T) {
	ctx := &ProcessorContext{
		Namespace: "monitoring",
	}

	list := []string{
		`<match kube-system>
       @type file
		 </match>`,
		`<match test.$thisns>
		   @type stdout
		 </match>`,
		`<match **>
		   @type rewrite_tag_filter
		 </match>
		`,
		`
		<source>
		  @type syslog
		</source>
		`,
	}

	for _, s := range list {
		fragment, err := fluentd.ParseString(s)
		assert.Nil(t, err)

		_, err = Process(fragment, ctx, &fixDestinations{})
		assert.NotNil(t, err)
	}
}
