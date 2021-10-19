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

// ReloadConfiguration talks to fluentd's RPC endpoont. If r is nil does nothing
func (r *Reloader) ReloadConfiguration() {
	if r == nil {
		logrus.Infof("Not reloading fluentd (fake or filesystem datasource used)")
		return
	}

	resp, err := http.Post(fmt.Sprintf("http://127.0.0.1:%d/api/config.reload", r.port), "application/json", nil)
	if err != nil {
		logrus.Errorf("fluentd config.reload post request failed: %+v", err)
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		body, _ := ioutil.ReadAll(resp.Body)
		logrus.Errorf("fluentd config.reload endpoint returned statuscode %v; response: %v", resp.StatusCode, string(body))
	}
}
