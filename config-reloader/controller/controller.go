// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package controller

import (
	"time"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/generator"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	Interval   time.Duration
	OutputDir  string
	Reloader   *fluentd.Reloader
	Datasource datasource.Datasource
	Generator  *generator.Generator
}

func (c *Controller) Run(stop <-chan struct{}) {
	for {
		err := c.RunOnce()
		if err != nil {
			logrus.Error(err)
		}

		select {
		case <-time.After(c.Interval):
		case <-stop:
			logrus.Info("Terminating main controller loop")
			return
		}
	}
}

// New creates new controller
func New(cfg *config.Config) (*Controller, error) {
	var ds datasource.Datasource
	var err error
	var reloader *fluentd.Reloader

	if cfg.Datasource == "fake" {
		ds = datasource.NewFakeDatasource()
	} else if cfg.Datasource == "fs" {
		ds = datasource.NewFileSystemDatasource(cfg.FsDatasourceDir, cfg.OutputDir)
	} else {
		ds, err = datasource.NewKubernetesDatasource(cfg)
		if err != nil {
			return nil, err
		}
		reloader = fluentd.NewReloader(cfg.FluentdRPCPort)
	}

	gen := generator.New(cfg)
	gen.SetStatusUpdater(ds)

	return &Controller{
		Interval:   time.Duration(cfg.IntervalSeconds) * time.Second,
		OutputDir:  cfg.OutputDir,
		Reloader:   reloader,
		Datasource: ds,
		Generator:  gen,
	}, nil
}

func (c *Controller) RunOnce() error {
	logrus.Infof("Running main control loop")

	allNamespaces, err := c.Datasource.GetNamespaces()
	if err != nil {
		return err
	}

	c.Generator.SetModel(allNamespaces)
	configHashes, err := c.Generator.RenderToDisk(c.OutputDir)
	if err != nil {
		return nil
	}

	needsReload := false

	for _, nsConfig := range allNamespaces {
		newHash, found := configHashes[nsConfig.Name]
		if !found {
			logrus.Infof("No config updates for namespace %s", nsConfig.Name)
			// error rendering config for the namespace, skip
			continue
		}

		if newHash != nsConfig.PreviousConfigHash {
			needsReload = true
			c.Datasource.WriteCurrentConfigHash(nsConfig.Name, newHash)
		}
	}

	if needsReload {
		c.Reloader.ReloadConfiguration()
	}

	return nil
}
