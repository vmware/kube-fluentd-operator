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
	Updater    Updater
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
		case <-c.Updater.GetUpdateChannel():
		case <-stop:
			logrus.Info("Terminating main controller loop")
			return
		}
	}
}

// New creates new controller
func New(cfg *config.Config) (*Controller, error) {
	var ds datasource.Datasource
	var up Updater
	var err error
	var reloader *fluentd.Reloader

	if cfg.Datasource == "fake" {
		ds = datasource.NewFakeDatasource()
		up = NewFixedTimeUpdater(cfg.IntervalSeconds)
	} else if cfg.Datasource == "fs" {
		ds = datasource.NewFileSystemDatasource(cfg.FsDatasourceDir, cfg.OutputDir)
		up = NewFixedTimeUpdater(cfg.IntervalSeconds)
	} else {
		updateChan := make(chan time.Time, 1)
		ds, err = datasource.NewKubernetesInformerDatasource(cfg, updateChan)
		if err != nil {
			return nil, err
		}
		reloader = fluentd.NewReloader(cfg.FluentdRPCPort)
		up = NewOnDemandUpdater(updateChan)
	}

	gen := generator.New(cfg)
	gen.SetStatusUpdater(ds)

	return &Controller{
		Updater:    up,
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

	c.Generator.CleanupUnusedFiles(c.OutputDir, configHashes)

	return nil
}
