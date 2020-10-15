// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/controller"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/metrics"

	"github.com/sirupsen/logrus"
)

func main() {
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
		validator := fluentd.NewValidator(cfg.FluentdValidateCommand)
		if err := validator.EnsureUsable(); err != nil {
			logrus.Fatalf("Bad validate command used: '%s', either use correct one or none at all: %+v",
				cfg.FluentdValidateCommand, err)
		}
	}

	logrus.SetLevel(cfg.GetLogLevel())

	ctrl, err := controller.New(cfg)
	if err != nil {
		logrus.Fatalf("Cannot start control loop %+v", err)
	}

	if cfg.IntervalSeconds == 0 {
		ctrl.RunOnce()
		return
	}

	stopChan := make(chan struct{}, 1)
	go handleSigterm(stopChan)

	if cfg.PrometheusEnabled {
		metrics.InitMetrics(cfg.MetricsPort)
	}

	ctrl.Run(stopChan)
}

func handleSigterm(stopChan chan struct{}) {
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	sig := <-signals
	logrus.Infof("Received %v. Terminating...", sig)
	close(stopChan)
}
