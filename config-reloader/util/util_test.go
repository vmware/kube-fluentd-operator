// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMakeFluentdSafeName(t *testing.T) {
	assert.Equal(t, "a", MakeFluentdSafeName("a"))
	assert.Equal(t, "123", MakeFluentdSafeName("123"))
	assert.Equal(t, "", MakeFluentdSafeName(""))
	assert.Equal(t, "a-a", MakeFluentdSafeName("a.a"))
	assert.Equal(t, "a-a", MakeFluentdSafeName("a\na"))
	assert.Equal(t, "---", MakeFluentdSafeName("   "))
}

func TestToRubyMapLiteral(t *testing.T) {
	assert.Equal(t, "{}", ToRubyMapLiteral(map[string]string{}))
	assert.Equal(t, "{'a'=>'1'}", ToRubyMapLiteral(map[string]string{
		"a": "1",
	}))
	assert.Equal(t, "{'a'=>'1','z'=>'2'}", ToRubyMapLiteral(map[string]string{
		"a": "1",
		"z": "2",
	}))
}

func TestTrim(t *testing.T) {
	assert.Equal(t, "a", Trim("a"))
	assert.Equal(t, "a", Trim("  a"))
	assert.Equal(t, "a", Trim("a  \t "))
	assert.Equal(t, "a", Trim(" \t a   "))
}

func TestTrimTrailingComment(t *testing.T) {
	assert.Equal(t, "a", TrimTrailingComment("a #12451345"))
	assert.Equal(t, "a", TrimTrailingComment("a"))
	assert.Equal(t, "a", TrimTrailingComment("a#########"))
}

func TestLabelsParseOk(t *testing.T) {
	inputs := map[string]map[string]string{
		"$labels(a=b,,,)":                  {"a": "b"},
		"$labels(a=1, b=2)":                {"a": "1", "b": "2"},
		"$labels(x=y,b=1)":                 {"b": "1", "x": "y"},
		"$labels(x=1, b = 1)":              {"b": "1", "x": "1"},
		"$labels(x=1, a=)":                 {"a": "", "x": "1"},
		"$labels(hello/world=ok, a=value)": {"hello/world": "ok", "a": "value"},
		"$labels(x=1, _container=main)":    {"_container": "main", "x": "1"},
	}

	for tag, result := range inputs {
		processed, err := ParseTagToLabels(tag)
		assert.Nil(t, err, "Got an error instead: %+v", err)
		assert.Equal(t, result, processed)
	}
}

func TestLabelsParseNotOk(t *testing.T) {
	inputs := []string{
		"$labels",
		"$labels()",
		"$labels(=)",
		"$labels(=f)",
		"$labels(.=*)",
		"$labels(a=.)",
		"$labels(a==1)",
		"$labels(-a=sfd)",
		"$labels(a=-sfd)",
		"$labels(a*=hello)",
		"$labels(a=*)",
		"$labels(a=1, =2)",
		"$labels(_container=)", // empty container name
		"$labels(app.kubernetes.io/name=*)",
	}

	for _, tag := range inputs {
		res, err := ParseTagToLabels(tag)
		assert.NotNil(t, err, "Got this instead for %s: %+v", tag, res)
	}
}
