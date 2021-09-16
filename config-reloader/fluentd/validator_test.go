// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

const validateCommand = "./fake-fluentd.sh -p plugins"

func TestValidConfigString(t *testing.T) {
	ctx := context.Background()

	s := `
	<match **>
	  @type null
	</match>
	`

	validator := NewValidator(ctx, validateCommand, 30*time.Second)

	err := validator.EnsureUsable()
	assert.Nil(t, err, "Must succeed but failed with: %+v", err)

	err = validator.ValidateConfigExtremely(s, "namespace-1")
	assert.Nil(t, err, "Must succeed but failed with %+v", err)
}

func TestUnusable(t *testing.T) {
	ctx := context.Background()

	validator := NewValidator(ctx, "./no-such command", 30*time.Second)

	err := validator.EnsureUsable()
	assert.NotNil(t, err, "Must have failed")
}

func TestBadConfigString(t *testing.T) {
	ctx := context.Background()

	s := `
	# ERROR <- this is a marker to cause failure
	<match **>
	  @type null
	</match>
	`

	validator := NewValidator(ctx, validateCommand, 30*time.Second)

	err := validator.EnsureUsable()
	assert.Nil(t, err, "Must succeed but failed with: %+v", err)

	err = validator.ValidateConfigExtremely(s, "namespace-1")
	assert.NotNil(t, err)
}
