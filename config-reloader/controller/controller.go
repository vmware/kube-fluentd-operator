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
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	Updater        Updater
	OutputDir      string
	Reloader       *fluentd.Reloader
	Datasource     datasource.Datasource
	Generator      *generator.Generator
	AllConfigsHash uint64
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

	allConfigNamespaces, err := c.Datasource.GetNamespaces(ctx)
	if err != nil {
		return err
	}

	c.Generator.SetModel(allConfigNamespaces)
	configHashes, err := c.Generator.RenderToDisk(ctx, c.OutputDir)
	if err != nil {
		return nil
	}

	needsReload := false

	logrus.Debugf("Config hashes returned in RunOnce loop: %v", configHashes)

	for _, nsConfig := range allConfigNamespaces {
		newHash, found := configHashes[nsConfig.Name]
		if !found {
			logrus.Infof("No config updates for namespace %s", nsConfig.Name)
			// error rendering config for the namespace, skip
			continue
		}

		if newHash != nsConfig.PreviousConfigHash {
			needsReload = true
			logrus.Debugf("Previous Config hash for ns %s is %v", nsConfig.Name, nsConfig.PreviousConfigHash)
			logrus.Debugf("New Config hash for ns %s is %v", nsConfig.Name, newHash)
			c.Datasource.WriteCurrentConfigHash(nsConfig.Name, newHash)
		}
	}

	// lastly, if number of all configs has changed, then need to reload configurations obviously!
	// this means a crd was deleted or reapplied, and GetNamespaces does not return it anymore
	// metahashing, hashing the object of hashes :)
	allConfigsHash, _ := util.MakeStructureHash(configHashes)
	if c.AllConfigsHash != allConfigsHash {
		needsReload = true
		c.AllConfigsHash = allConfigsHash
		logrus.Debugf("All Configs hash for all KFO is %v", c.AllConfigsHash)
	}

	if needsReload {
		c.Reloader.ReloadConfiguration()
	}

	c.Generator.CleanupUnusedFiles(c.OutputDir, configHashes)

	return nil
}
