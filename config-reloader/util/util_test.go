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
