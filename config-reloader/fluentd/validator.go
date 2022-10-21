// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/vmware/kube-fluentd-operator/config-reloader/util"

	"github.com/sirupsen/logrus"
)

// Validator validates a generated config using fluentd's --dry-run command
type Validator interface {
	ValidateConfig(config string, namespace string) error
	ValidateConfigExtremely(config string, namespace string) error
	EnsureUsable() error
}

type validatorState struct {
	command string
	args    []string
	timeout time.Duration
}

var justExitPluginDirective = `
# extreme validation
<source>
  @type just_exit
</source>
`

// NewValidator creates a Validator using the given command
func NewValidator(ctx context.Context, command string, timeout time.Duration) Validator {
	parts := strings.Split(util.Trim(command), " ")

	return &validatorState{
		command: parts[0],
		args:    parts[1:],
		timeout: timeout,
	}
}

func (v *validatorState) ValidateConfigExtremely(config string, namespace string) error {
	if v == nil {
		return nil
	}

	tmpfile, err := ioutil.TempFile("", "validate-ext-"+namespace)
	if err != nil {
		logrus.Errorf("error creating temporary file: %s", err.Error())
		return err
	}
	defer os.Remove(tmpfile.Name())

	config += justExitPluginDirective
	if _, err = tmpfile.WriteString(config); err != nil {
		logrus.Errorf("error writing config to temp file: %s", err.Error())
		return err
	}

	if err := tmpfile.Close(); err != nil {
		logrus.Errorf("error closing temp file: %s", err.Error())
		return err
	}

	args := make([]string, len(v.args))
	copy(args, v.args)

	args = append(args, "-q", "--no-supervisor", "-c", tmpfile.Name())

	out, err := util.ExecAndGetOutput(v.command, v.timeout, args...)

	// strip color stuff from fluentd output
	out = strings.TrimFunc(out, func(r rune) bool {
		return !unicode.IsPrint(r)
	})

	logrus.Debugf("Checked config for namespace %s with fluentd and got: %s", namespace, out)
	if err != nil {
		logrus.Errorf("error running validation command: %s", err.Error())
		return errors.New(out)
	}

	return nil
}

func (v *validatorState) ValidateConfig(config string, namespace string) error {
	if v == nil {
		return nil
	}

	tmpfile, err := ioutil.TempFile("", "validate-"+namespace)
	if err != nil {
		logrus.Errorf("error creating temporary file: %s", err.Error())
		return err
	}
	defer os.Remove(tmpfile.Name())

	if _, err = tmpfile.WriteString(config); err != nil {
		logrus.Errorf("error writing config to temp file: %s", err.Error())
		return err
	}

	if err := tmpfile.Close(); err != nil {
		logrus.Errorf("error closing temp file: %s", err.Error())
		return err
	}

	args := make([]string, len(v.args))
	copy(args, v.args)

	args = append(args, "--dry-run", "-c", tmpfile.Name())

	out, err := util.ExecAndGetOutput(v.command, v.timeout, args...)

	// strip color stuf from fluentd output
	out = strings.TrimFunc(out, func(r rune) bool {
		return !unicode.IsPrint(r)
	})

	logrus.Debugf("Checked config for namespace %s with fluentd and got: %s", namespace, out)
	if err != nil {
		logrus.Errorf("error running command: %s", err.Error())
		return errors.New(out)
	}

	return nil
}

func (v *validatorState) EnsureUsable() error {
	if v == nil {
		return nil
	}
	out, err := util.ExecAndGetOutput(v.command, v.timeout, "--version")
	if err != nil {
		return fmt.Errorf("invalid fluentd binary used %s: %+v", v.command, err)
	}

	logrus.Infof("Validator using %s at version %s", v.command, util.Trim(out))
	return nil
}
