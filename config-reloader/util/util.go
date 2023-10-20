// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package util

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"time"
	"unicode"

	"github.com/sirupsen/logrus"
)

const (
	maskFile       = 0664
	maskDirectory  = 0775
	MacroLabels    = "$labels"
	ContainerLabel = "_container"
)

var reValidLabelName = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9\/_.]*)?[A-Za-z0-9]$`)
var reValidLabelValue = regexp.MustCompile(`^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$`)

func Trim(s string) string {
	return strings.TrimSpace(s)
}

func MakeFluentdSafeName(s string) string {
	buf := &bytes.Buffer{}
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			buf.WriteRune('-')
		} else {
			buf.WriteRune(r)
		}
	}

	return buf.String()
}

func ToRubyMapLiteral(labels map[string]string) string {
	if len(labels) == 0 {
		return "{}"
	}

	buf := &bytes.Buffer{}
	buf.WriteString("{")
	for _, k := range SortedKeys(labels) {
		fmt.Fprintf(buf, "'%s'=>'%s',", k, labels[k])
	}
	buf.Truncate(buf.Len() - 1)
	buf.WriteString("}")

	return buf.String()
}

func Hash(owner string, value string) string {
	h := sha256.New()

	h.Write([]byte(owner))
	h.Write([]byte(":"))
	h.Write([]byte(value))

	b := h.Sum(nil)
	return hex.EncodeToString(b[0:20])
}

func SortedKeys(m map[string]string) []string {
	keys := make([]string, len(m))
	i := 0

	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)

	return keys
}

// ExecAndGetOutput exec and returns output of the command if timeout then kills the process and returns error
func ExecAndGetOutput(cmd string, timeout time.Duration, args ...string) (string, error) {
	c := exec.Command(cmd, args...)
	var b bytes.Buffer
	c.Stdout = &b
	c.Stderr = &b
	var err error
	if err = c.Start(); err != nil {
		out := b.Bytes()
		return string(out), err
	}

	// Wait for the process to finish or kill it after a timeout (whichever happens first):
	done := make(chan error, 1)
	go func() {
		done <- c.Wait()
	}()

	select {
	case <-time.After(timeout):
		if err = c.Process.Kill(); err != nil {
			err = fmt.Errorf("process killed as timeout reached after %s, but kill failed with err: %s", timeout, err.Error())
		} else {
			err = fmt.Errorf("process killed as timeout reached after %s", timeout)
		}
	case err = <-done:
	}
	out := b.Bytes()

	return string(out), err
}

func WriteStringToFile(filename string, data string) error {
	return os.WriteFile(filename, []byte(data), maskFile)
}

func TrimTrailingComment(line string) string {
	i := strings.IndexByte(line, '#')
	if i > 0 {
		line = Trim(line[0:i])
	} else {
		line = Trim(line)
	}

	return line
}

func ParseTagToLabels(tag string) (map[string]string, error) {
	if !strings.HasPrefix(tag, MacroLabels+"(") &&
		!strings.HasSuffix(tag, ")") {
		return nil, fmt.Errorf("bad $labels macro use: %s", tag)
	}

	labelsOnly := tag[len(MacroLabels)+1 : len(tag)-1]

	result := map[string]string{}

	records := strings.Split(labelsOnly, ",")
	for _, rec := range records {
		if rec == "" {
			// be generous
			continue
		}
		kv := strings.Split(rec, "=")
		if len(kv) != 2 {
			return nil, fmt.Errorf("bad label definition: %s", kv)
		}

		k := Trim(kv[0])
		if k != ContainerLabel {
			if !reValidLabelName.MatchString(k) {
				return nil, fmt.Errorf("bad label name: %s", k)
			}
		}

		v := Trim(kv[1])
		if !reValidLabelValue.MatchString(v) {
			return nil, fmt.Errorf("bad label value: %s", v)
		}
		if k == ContainerLabel && v == "" {
			return nil, fmt.Errorf("value for %s cannot be empty string", ContainerLabel)
		}

		result[k] = v
	}

	if len(result) == 0 {
		return nil, errors.New("at least one label must be given")
	}

	return result, nil
}

func Match(labels map[string]string, contLabels map[string]string, contName string) bool {
	for k, v := range labels {
		value := contLabels[k]
		if k == "_container" {
			value = contName
		}

		if v != value {
			return false
		}
	}
	return true
}

func EnsureDirExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err = os.Mkdir(dir, maskDirectory)
		if err != nil {
			logrus.Errorln("Unexpected error occurred with output config directory: ", dir)
			return err
		}
	}
	return nil
}

func TemplateAndWriteFile(tmpl *template.Template, model interface{}, dest string) (err error) {
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, model)
	if err != nil {
		logrus.Warnf("Error rendering template file %s: %+v", dest, err)
		return nil
	}

	err = WriteStringToFile(dest, buf.String())
	if err != nil {
		return err
	}
	return nil
}
