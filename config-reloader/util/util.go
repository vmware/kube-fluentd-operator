// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package util

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"sort"
	"strings"
	"time"
	"unicode"
)

const (
	maskFile = 0664
)

func Trim(s string) string {
	return strings.TrimFunc(s, unicode.IsSpace)
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
	h := sha1.New()

	h.Write([]byte(owner))
	h.Write([]byte(":"))
	h.Write([]byte(value))

	b := h.Sum(nil)
	return hex.EncodeToString(b[:])
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

func ExecAndGetOutput(cmd string, args ...string) (string, error) {
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
	case <-time.After(30 * time.Second):
		if err = c.Process.Kill(); err != nil {
			err = fmt.Errorf("process killed as timeout reached after 10s,but kill failed with err:%s",err.Error())
		} else {
			err = errors.New("process killed as timeout reached after 10s")
		}
	case err = <-done:
	}
	out := b.Bytes()
	return string(out), err
}

func WriteStringToFile(filename string, data string) error {
	return ioutil.WriteFile(filename, []byte(data), maskFile)
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
