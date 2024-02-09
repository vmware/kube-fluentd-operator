// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/sirupsen/logrus"
)

type RPCMethod string

const (
	gracefulReloadConf RPCMethod = "config.gracefulReload"
	reloadConf         RPCMethod = "config.reload"
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

	logrus.Infof("Reloading fluentd configuration via /api/%s", gracefulReloadConf)
	if err := r.rpc(gracefulReloadConf); err != nil {
		logrus.Warnf("graceful reload failed: %+v", err)
		logrus.Infof("Reloading fluentd configuration via /api/%s", reloadConf)
		if err := r.rpc(reloadConf); err != nil {
			logrus.Error(err.Error())
		}
	}
}

// rpc calls the given fluentd HTTP RPC endpoint
// for more details see: https://docs.fluentd.org/deployment/rpc
func (r *Reloader) rpc(method RPCMethod) error {
	resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/%s", r.port, method))
	if err != nil {
		return fmt.Errorf("fluentd %s request failed: %w", method, err)
	}

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return fmt.Errorf("fluentd %s endpoint returned statuscode %v; response: %v", method, resp.StatusCode, string(body))
	}

	return nil
}
