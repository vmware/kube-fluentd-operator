// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseErrors(t *testing.T) {
	badInputs := []string{
		`
		@type hello
		`,
		`
		 </match>`,
		`
		</match>
		`,
		`<source>`,
		`<match>
		</filter>`,
	}

	for _, input := range badInputs {
		_, err := ParseString(input)
		assert.NotNil(t, err)
	}
}

func TestParse1(t *testing.T) {
	var s1 = `
	# hello
	<match {abc,def}>
		ms    1
		one_space 2
		no_value
		trailing   
		trailing_val val  
		# comment
		#inl_comment_val 123 # this is the comment
		#inl_comment  # this is the comment
	</match>
	`
	fragment, err := ParseString(s1)

	assert.Nil(t, err)
	assert.NotNil(t, fragment)
	fmt.Printf("%s", fragment)

	d := fragment[0]
	assert.Equal(t, "match", d.Name)
	assert.Equal(t, "{abc,def}", d.Tag)
	assert.Equal(t, "1", d.Params["ms"].Value)
	assert.Equal(t, "2", d.Params["one_space"].Value)
	assert.Equal(t, "", d.Params["no_value"].Value)
	assert.Equal(t, "", d.Params["trailing"].Value)
	assert.Equal(t, "val", d.Params["trailing_val"].Value)

	// TODO support inline comments
	//assert.Equal(t, "", d.Params["inl_comment"].Value)
	//assert.Equal(t, "123", d.Params["inl_comment_val"].Value)
}

func TestParseNestedToString(t *testing.T) {
	var nested = `
	<filter   myapp.access  >
		@type test # inline comment
		<record>
			host_param "#{Socket.gethostname}"
		</record>
	</filter>
	`

	fragment, err := ParseString(nested)
	assert.Nil(t, err)
	fmt.Printf("%s", fragment)
	s := fragment.String()

	assert.Equal(t,
		`<filter myapp.access>
  @type test

  <record>
    host_param "#{Socket.gethostname}"
  </record>
</filter>

`, s)
}

func TestGetType(t *testing.T) {
	var nested = `
	# http://this.host:9880/myapp.access?json={"event":"data"}
	<source>
		hello http
		port 9880
	</source>
	`

	fragment, err := ParseString(nested)
	assert.Nil(t, err)
	fmt.Printf("%s", fragment)

	src := fragment[0]
	assert.Equal(t, "", src.Type())
}
func TestParseNested(t *testing.T) {
	var nested = `
	# http://this.host:9880/myapp.access?json={"event":"data"}
	<source>
		@type http
		port 9880
	</source>
	
	<filter myapp.access>
		type record_transformer
		<record>
			host_param "#{Socket.gethostname}"
		</record>
	</filter>
	
	<match myapp.access>
		@type file
		path /var/log/fluent/access
	</match>
	`

	fragment, err := ParseString(nested)
	assert.Nil(t, err)
	fmt.Printf("%s", fragment)

	src := fragment[0]
	assert.Equal(t, "source", src.Name)
	assert.Equal(t, "http", src.Params["@type"].Value)
	assert.Equal(t, "9880", src.Param("port"))
	assert.Equal(t, "", src.Param("no-such-param"))
	assert.Equal(t, "http", src.Type())

	filter := fragment[1]
	assert.Equal(t, "filter", filter.Name)
	assert.Equal(t, "myapp.access", filter.Tag)

	record := filter.Nested[0]
	assert.Equal(t, "record", record.Name)
	assert.Equal(t, "\"#{Socket.gethostname}\"", record.Param("host_param"))

	match := fragment[2]
	assert.Equal(t, "match", match.Name)
	assert.Equal(t, "file", match.Type())
	assert.Equal(t, "/var/log/fluent/access", match.Param("path"))
}
