// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package controller

import (
	"context"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/generator"

	"github.com/sirupsen/logrus"
)

type Controller interface {
	Run(ctx context.Context, stop <-chan struct{})
	RunOnce(ctx context.Context) error
	GetTotalConfigNS() int
}

type controllerInstance struct {
	Updater          Updater
	Reloader         *fluentd.Reloader
	Datasource       datasource.Datasource
	Generator        generator.Generator
	outputDir        string
	numTotalConfigNS int
}

var _ Controller = &controllerInstance{}

// New creates new controller
func New(ctx context.Context, cfg *config.Config, ds datasource.Datasource, up Updater) (Controller, error) {
	var reloader *fluentd.Reloader
	gen := generator.New(ctx, cfg)
	gen.SetStatusUpdater(ctx, ds)

	switch cfg.Datasource {
	case "fake", "fs":
		logrus.Infof("Setting reloader to null because is running locally")
	default:
		reloader = fluentd.NewReloader(ctx, cfg.FluentdRPCPort)
	}

	return &controllerInstance{
		Updater:    up,
		Reloader:   reloader,
		Datasource: ds,
		Generator:  gen,
		outputDir:  cfg.OutputDir,
	}, nil
}

func (c *controllerInstance) RunOnce(ctx context.Context) error {
	logrus.Infof("Running main control loop")

	allConfigNamespaces, err := c.Datasource.GetNamespaces(ctx)
	if err != nil {
		return err
	}

	c.Generator.SetModel(allConfigNamespaces)
	configHashes, err := c.Generator.RenderToDisk(ctx, c.outputDir)
	if err != nil {
		return nil
	}

	needsReload := false

	logrus.Infof("Config hashes returned in RunOnce loop: %v", configHashes)

	for _, nsConfig := range allConfigNamespaces {
		logrus.Debugf("Comparing hash with previous one for namespace: %v", nsConfig.Name)
		newHash, found := configHashes[nsConfig.Name]
		if !found {
			logrus.Infof("No config updates for namespace %s", nsConfig.Name)
			// error rendering config for the namespace, skip
			continue
		}

		if newHash != nsConfig.PreviousConfigHash {
			logrus.Infof("Detecting updates for namespace %s", nsConfig.Name)
			needsReload = true
			c.Datasource.WriteCurrentConfigHash(nsConfig.Name, newHash)
		}
	}

	// lastly, if number of configs has changed, then need to reload configurations obviously!
	// this means a crd was deleted or reapplied, and GetNamespaces does not return it anymore
	if c.numTotalConfigNS != len(allConfigNamespaces) {
		logrus.Infof("New namespaces found. Reloading fluentd...")
		needsReload = true
		c.numTotalConfigNS = len(allConfigNamespaces)
	}

	if needsReload {
		c.Reloader.ReloadConfiguration()
	}

	c.Generator.CleanupUnusedFiles(c.outputDir, configHashes)

	return nil
}

func (c *controllerInstance) Run(ctx context.Context, stop <-chan struct{}) {
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

func (c *controllerInstance) GetTotalConfigNS() int {
	return c.numTotalConfigNS
}
