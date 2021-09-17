// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/controller"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/metrics"

	"github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()
	cfg := &config.Config{}

	if err := cfg.ParseFlags(os.Args[1:]); err != nil {
		logrus.Fatalf("flag parsing error: %v", err)
	}
	logrus.Infof("Version: %s", config.Version)

	logrus.Infof("Config: %+v", cfg)

	if err := cfg.Validate(); err != nil {
		logrus.Fatalf("Config validation failed: %+v", err)
	}

	if cfg.FluentdValidateCommand != "" {
		validator := fluentd.NewValidator(ctx, cfg.FluentdValidateCommand, time.Second*time.Duration(cfg.ExecTimeoutSeconds))
		if err := validator.EnsureUsable(); err != nil {
			logrus.Fatalf("Bad validate command used: '%s', either use correct one or none at all: %+v",
				cfg.FluentdValidateCommand, err)
		}
	}

	logrus.SetLevel(cfg.GetLogLevel())

	ctrl, err := controller.New(ctx, cfg)
	if err != nil {
		logrus.Fatalf("Cannot start control loop %+v", err)
	}

	// Add this for a timeout between 0-120 seconds (default: 30 (ExecTimeoutSeconds))
	// This is for golang/fluentd race condition when KFO starts/restarts:
	if cfg.ExecTimeoutSeconds > 0 && cfg.ExecTimeoutSeconds <= 120 {
		logrus.Infof("Sleeping for %v seconds in order for fluentd to be ready.", cfg.ExecTimeoutSeconds)
		time.Sleep(time.Second * time.Duration(cfg.ExecTimeoutSeconds))
	}

	if cfg.IntervalSeconds == 0 {
		ctrl.RunOnce(ctx)
		return
	}

	stopChan := make(chan struct{}, 1)
	go handleSigterm(stopChan)

	if cfg.PrometheusEnabled {
		metrics.InitMetrics(cfg.MetricsPort)
	}

	ctrl.Run(ctx, stopChan)
}

func handleSigterm(stopChan chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	sig := <-signals
	logrus.Infof("Received %v. Terminating...", sig)
	close(stopChan)
}
