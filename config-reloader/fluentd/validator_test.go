// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const validateCommand = "./fake-fluentd.sh -p plugins"

func TestValidConfigString(t *testing.T) {
	s := `
	<match **>
	  @type null
	</match>
	`

	validator := NewValidator(validateCommand)

	err := validator.EnsureUsable()
	assert.Nil(t, err, "Must succeed but failed with: %+v", err)

	err = validator.ValidateConfig(s, "namespace-1")
	assert.Nil(t, err, "Must succeed but failed with %+v", err)
}

func TestUnusable(t *testing.T) {
	validator := NewValidator("./no-such command")

	err := validator.EnsureUsable()
	assert.NotNil(t, err, "Must have failed")
}

func TestBadConfigString(t *testing.T) {
	s := `
	# ERROR <- this is a marker to cause failure
	<match **>
	  @type null
	</match>
	`

	validator := NewValidator(validateCommand)

	err := validator.EnsureUsable()
	assert.Nil(t, err, "Must succeed but failed with: %+v", err)

	err = validator.ValidateConfig(s, "namespace-1")
	assert.NotNil(t, err)
}
