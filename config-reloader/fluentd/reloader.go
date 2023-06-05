// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/sirupsen/logrus"
)

// Reloader sends a reload signal to fluentd
type Reloader struct {
	port int
}

// NewReloader will notify on the given rpc port
func NewReloader(ctx context.Context, port int) *Reloader {
	return &Reloader{
		port: port,
	}
}

// ReloadConfiguration talks to fluentd's RPC endpoint. If r is nil does nothing
func (r *Reloader) ReloadConfiguration() {
	if r == nil {
		logrus.Infof("Not reloading fluentd (fake or filesystem datasource used)")
		return
	}

	logrus.Debugf("Reloading fluentd configuration gracefully via POST to /api/config.gracefulReload")

	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/config.gracefulReload", r.port))
	if err != nil {
		logrus.Errorf("fluentd config.gracefulReload request failed: %+v", err)
	} else if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("fluentd config.gracefulReload endpoint returned statuscode %v; response: %v", resp.StatusCode, string(body))
	}
}
