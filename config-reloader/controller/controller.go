// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package controller

import (
	"context"
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

func (c *Controller) Run(ctx context.Context, stop <-chan struct{}) {
	for {
		err := c.RunOnce(ctx)
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
func New(ctx context.Context, cfg *config.Config) (*Controller, error) {
	var ds datasource.Datasource
	var up Updater
	var err error
	var reloader *fluentd.Reloader

	switch cfg.Datasource {
	case "fake":
		ds = datasource.NewFakeDatasource(ctx)
		up = NewFixedTimeUpdater(ctx, cfg.IntervalSeconds)
	case "fs":
		ds = datasource.NewFileSystemDatasource(ctx, cfg.FsDatasourceDir, cfg.OutputDir)
		up = NewFixedTimeUpdater(ctx, cfg.IntervalSeconds)
	default:
		updateChan := make(chan time.Time, 1)
		ds, err = datasource.NewKubernetesInformerDatasource(ctx, cfg, updateChan)
		if err != nil {
			return nil, err
		}
		reloader = fluentd.NewReloader(ctx, cfg.FluentdRPCPort)
		up = NewOnDemandUpdater(ctx, updateChan)
	}

	gen := generator.New(ctx, cfg)
	gen.SetStatusUpdater(ctx, ds)

	return &Controller{
		Updater:    up,
		OutputDir:  cfg.OutputDir,
		Reloader:   reloader,
		Datasource: ds,
		Generator:  gen,
	}, nil
}

func (c *Controller) RunOnce(ctx context.Context) error {
	logrus.Infof("Running main control loop")

	allNamespaces, err := c.Datasource.GetNamespaces(ctx)
	if err != nil {
		return err
	}

	c.Generator.SetModel(allNamespaces)
	configHashes, err := c.Generator.RenderToDisk(ctx, c.OutputDir)
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
