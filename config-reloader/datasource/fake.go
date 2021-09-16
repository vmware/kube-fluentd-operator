// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package datasource

import (
	"context"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

var template = `
<match **>
  @type logzio_buffered
  endpoint_url https://listener.logz.io:8071?token=secret
  output_include_time true
  output_include_tags true
  buffer_type    file
  buffer_path    /var/log/logzio-$my_ns.buffer
  flush_interval 10s
  buffer_chunk_limit 1m
</match>
`

type fakeDatasource struct {
	hashes map[string]string
}

func makeFakeConfig(namespace string) string {
	contents := template
	contents = strings.ReplaceAll(contents, "$ns$", namespace)
	contents = strings.ReplaceAll(contents, "$ts$", time.Now().String())

	return contents
}

func (d *fakeDatasource) GetNamespaces(ctx context.Context) ([]*NamespaceConfig, error) {
	res := []*NamespaceConfig{}

	for _, ns := range []string{"kube-system", "monitoring", "csp-main"} {
		res = append(res, &NamespaceConfig{
			Name:          ns,
			FluentdConfig: makeFakeConfig(ns),
		})
	}

	// unconfigured namespace
	res = append(res, &NamespaceConfig{
		Name: "not-configured",
		FluentdConfig: `
		<match **>
		  @type null
		</match>
		`,
	})
	return res, nil
}

func (d *fakeDatasource) WriteCurrentConfigHash(namespace string, hash string) {
	d.hashes[namespace] = hash
}

func (d *fakeDatasource) UpdateStatus(ctx context.Context, namespace string, status string) {
	logrus.Infof("Setting status of namespace %s to %s", namespace, status)
}

// NewFakeDatasource returns a predefined set of namespaces + configs
func NewFakeDatasource(ctx context.Context) Datasource {
	return &fakeDatasource{
		hashes: make(map[string]string),
	}
}
