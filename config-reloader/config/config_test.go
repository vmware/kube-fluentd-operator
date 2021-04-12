// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBadConfigs(t *testing.T) {
	inputs := [][]string{
		{"--datasource", "fs"},
		{"--id", "???"},
		{"--log-level", "hobbit"},
		{"--fluentd-loglevel", "hobbit"},
		{"--annotation", "|kl"},
		{"--status-annotation", "/hello"},
		{"--meta-key=test"},
		{"--meta-key=test", "--meta-values="},
		{"--meta-key=test", "--meta-values=."},
		{"--meta-key=test", "--meta-values='"},
		{"--meta-key=t''t", "--meta-values="},
		{"--meta-key=test", "--meta-values=a"},
		{"--meta-key=test", "--meta-values=a="},
		{"--meta-key=test", "--meta-values=a=="},
	}

	for _, args := range inputs {
		cfg := &Config{}
		err := cfg.ParseFlags(args)
		assert.Nil(t, err)

		err = cfg.Validate()
		assert.NotNil(t, err, "'%v' must fail validation", args)
		fmt.Printf("error %s\n", err)
	}
}

func TestNormalization(t *testing.T) {
	cfg := &Config{}
	err := cfg.ParseFlags([]string{"--interval=-1"})
	assert.Nil(t, err)
	err = cfg.Validate()
	assert.Nil(t, err)

	assert.Equal(t, 60, cfg.IntervalSeconds)
	assert.Equal(t, "info", cfg.LogLevel)
}
