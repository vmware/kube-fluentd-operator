// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package generator

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"text/template"

	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
	"github.com/vmware/kube-fluentd-operator/config-reloader/processors"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"

	"github.com/sirupsen/logrus"
)

const (
	mainConfigFile = "fluent.conf"
	maskDirectory  = 0775
)

// Generator produces fluentd config files
type Generator struct {
	templatesDir string
	model        []*datasource.NamespaceConfig
	cfg          *config.Config
	validator    fluentd.Validator
	su           datasource.StatusUpdater
}

func ensureDirExists(dir string) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		_ = os.Mkdir(dir, maskDirectory)
	}
}

func (g *Generator) makeNamespaceConfiguration(ns *datasource.NamespaceConfig) (string, error) {
	// unconfigured namespace
	if ns.FluentdConfig == "" {
		return "", nil
	}

	fragment, err := fluentd.ParseString(ns.FluentdConfig)
	if err != nil {
		return "", err
	}

	ctx := &processors.ProcessorContext{
		Namepsace:    ns.Name,
		AllowFile:    g.cfg.AllowFile,
		DeploymentID: g.cfg.ID,
	}

	fragment, err = processors.Apply(fragment, ctx, processors.DefaultProcessors()...)
	if err != nil {
		return "", err
	}

	return fragment.String(), nil
}

func (g *Generator) renderMainFile(mainFile string, outputDir string, dest string) (map[string]string, error) {
	tmpl, err := template.New(filepath.Base(mainFile)).ParseFiles(mainFile)
	if err != nil {
		return nil, err
	}

	fileHashesByNs := map[string]string{}

	newFiles := []string{}
	model := struct {
		KubeSystem bool
		Namespaces []string
		MetaKey    string
		MetaValue  string
	}{}

	if g.cfg.MetaKey != "" {
		model.MetaKey = g.cfg.MetaKey

		buf := &bytes.Buffer{}
		buf.WriteString("{")
		for k, v := range g.cfg.ParsedMetaValues {
			buf.WriteString(fmt.Sprintf("'%s' => '%s', ", k, v))
		}
		buf.Truncate(buf.Len() - 2)
		buf.WriteString("}")

		model.MetaValue = buf.String()
	}

	for _, nsConf := range g.model {
		if nsConf.Name == "kube-system" {
			model.KubeSystem = true
			fileHashesByNs["kube-system"] = util.Hash("", nsConf.FluentdConfig)

			// don't validate the kube-system, just render it
			err = util.WriteStringToFile(filepath.Join(outputDir, "kube-system.conf"), nsConf.FluentdConfig)
			if err != nil {
				logrus.Infof("Cannot store config file for namespace %s", nsConf.Name)
			}
			continue
		}

		// render config
		renderedConfig, err := g.makeNamespaceConfiguration(nsConf)
		configHash := util.Hash("", renderedConfig)
		if err != nil {
			configHash = util.Hash("ERROR", err.Error())
		}

		if err != nil {
			logrus.Infof("Configuration for namespace %s cannot be validated: %+v", nsConf.Name, err)
			if nsConf.PreviousConfigHash != configHash {
				g.updateStatus(nsConf.Name, err.Error())
			}
			fileHashesByNs[nsConf.Name] = configHash
			continue
		}

		// namespae is not configured
		if renderedConfig == "" {
			fileHashesByNs[nsConf.Name] = configHash
			if nsConf.PreviousConfigHash != configHash && nsConf.IsKnownFromBefore {
				// empty config is a valid input, clear error status
				g.updateStatus(nsConf.Name, "")
			}
			continue
		}

		if nsConf.PreviousConfigHash != configHash && g.validator != nil {
			err = g.validator.ValidateConfig(renderedConfig, nsConf.Name)
			if err != nil {
				logrus.Infof("Configuration for namespace %s cannot be validated with fluentd: %+v", nsConf.Name, err)
				if nsConf.PreviousConfigHash != configHash {
					// only update status if error caused by different input
					g.updateStatus(nsConf.Name, err.Error())
				}
				fileHashesByNs[nsConf.Name] = configHash
				continue
			}
		}

		filename := fmt.Sprintf("ns-%s.conf", nsConf.Name)
		newFiles = append(newFiles, filename)
		fileHashesByNs[nsConf.Name] = configHash
		err = util.WriteStringToFile(filepath.Join(outputDir, filename), renderedConfig)
		if err != nil {
			logrus.Infof("Cannot store config file for namespace %s", nsConf.Name)
		}

		if nsConf.PreviousConfigHash != configHash {
			// clear error
			g.updateStatus(nsConf.Name, "")
		}
	}

	model.Namespaces = newFiles
	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, model)

	err = util.WriteStringToFile(dest, buf.String())
	if err != nil {
		return nil, err
	}

	return fileHashesByNs, nil
}

func (g *Generator) updateStatus(namespace string, status string) {
	g.su.UpdateStatus(namespace, status)
}

func (g *Generator) renderIncludableFile(templateFile string, dest string) {
	tmpl, err := template.New(filepath.Base(templateFile)).ParseFiles(templateFile)
	if err != nil {
		logrus.Warnf("Error processing template file %s: %+v", templateFile, err)
		return
	}

	// this is the model for the includable files
	ctx := struct {
		ID string
	}{
		ID: util.MakeFluentdSafeName(g.cfg.ID),
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, ctx)
	if err != nil {
		logrus.Warnf("Error rendering template file %s: %+v", templateFile, err)
		return
	}

	util.WriteStringToFile(dest, buf.String())
}

// RenderToDisk write only valid configurations to disk
func (g *Generator) RenderToDisk(outputDir string) (map[string]string, error) {
	ensureDirExists(outputDir)
	outputDir, _ = filepath.Abs(outputDir)
	res := map[string]string{}

	files, err := filepath.Glob(fmt.Sprintf("%s/*.conf", g.templatesDir))
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		base := filepath.Base(f)
		targetDest := path.Join(outputDir, base)

		if base != mainConfigFile {
			g.renderIncludableFile(f, targetDest)
		} else {
			res, err = g.renderMainFile(f, outputDir, targetDest)
			if err != nil {
				logrus.Warnf("Cannot write main file %s: %+v", f, err)
				return nil, err
			}
		}
	}

	return res, nil
}

// SetModel stores the model for later
func (g *Generator) SetModel(model []*datasource.NamespaceConfig) {
	g.model = model
}

// SetStatusUpdater configures a statusUpdater for later. nil updater is fine
func (g *Generator) SetStatusUpdater(su datasource.StatusUpdater) {
	g.su = su
}

// New creates a default impl
func New(cfg *config.Config) *Generator {
	templatesDir, _ := filepath.Abs(cfg.TemplatesDir)
	var validator fluentd.Validator

	if cfg.FluentdValidateCommand != "" {
		validator = fluentd.NewValidator(cfg.FluentdValidateCommand)
	}

	return &Generator{
		templatesDir: templatesDir,
		cfg:          cfg,
		validator:    validator,
	}
}
