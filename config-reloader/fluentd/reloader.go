// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Reloader sends a reload signal to fluentd
type Reloader struct {
	port int
        path string
}

// NewReloader will notify on the given rpc port
func NewReloader(ctx context.Context, port int, path string) *Reloader {
	return &Reloader{
		port: port,
                path: path,
	}
}

// ReloadConfiguration talks to fluentd's RPC endpoont. If r is nil does nothing
func (r *Reloader) ReloadConfiguration() {
	if r == nil {
		logrus.Infof("Not reloading fluentd (fake or filesystem datasource used)")
		return
	}
	_, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d%s", r.port, r.path), "application/json", nil)

	if err != nil {
		logrus.Errorf("cannot notify fluentd: %+v", err)
	}
}
