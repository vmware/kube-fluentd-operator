// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package generator

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
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

	onlyProcess = 1
	onlyPrepare = 2
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

func (g *Generator) makeNamespaceConfiguration(ns *datasource.NamespaceConfig, genCtx *processors.GenerationContext, mode int) (string, string, error) {
	// unconfigured namespace
	if ns.FluentdConfig == "" {
		return "", "", nil
	}

	fragment, err := fluentd.ParseString(ns.FluentdConfig)
	if err != nil {
		return "", "", err
	}

	ctx := g.makeContext(ns, genCtx)

	if mode == onlyPrepare {
		prep, err := processors.Prepare(fragment, ctx, processors.DefaultProcessors()...)
		if err != nil {
			return "", "", err
		}

		return "", prep.String(), nil
	}

	if mode == onlyProcess {
		fragment, err = processors.Process(fragment, ctx, processors.DefaultProcessors()...)
		if err != nil {
			return "", "", err
		}
		return fragment.String(), "", nil
	}

	return "", "", fmt.Errorf("bad mode: %d", mode)
}

func extractPrepConfig(ns string, prepareConfigs map[string]interface{}) (string, error) {
	what, ok := prepareConfigs[ns]

	if !ok {
		return "", nil
	}

	switch what := what.(type) {
	case string:
		return what, nil
	case error:
		return "", what
	}

	return "", nil
}

func (g *Generator) renderMainFile(mainFile string, outputDir string, dest string) (map[string]string, error) {
	tmpl, err := template.New(filepath.Base(mainFile)).ParseFiles(mainFile)
	if err != nil {
		return nil, err
	}

	fileHashesByNs := map[string]string{}

	newFiles := []string{}
	model := struct {
		AdminNamespace          bool
		Namespaces              []string
		MetaKey                 string
		MetaValue               string
		PreprocessingDirectives []string
	}{}

	if g.cfg.MetaKey != "" {
		model.MetaKey = g.cfg.MetaKey
		model.MetaValue = util.ToRubyMapLiteral(g.cfg.ParsedMetaValues)
	}

	genCtx := &processors.GenerationContext{
		ReferencedBridges: map[string]bool{},
	}

	prepareConfigs := g.generatePrepareConfigs(genCtx)

	// process the admin namespace first to collect the virtual plugins
	for _, nsConf := range g.model {
		if nsConf.Name != g.cfg.AdminNamespace {
			continue
		}

		model.AdminNamespace = true

		fragment, err := fluentd.ParseString(nsConf.FluentdConfig)
		if err != nil {
			return nil, err
		}

		fragment = processors.ExtractPlugins(genCtx, fragment)

		// normalize system config
		renderedConfig := fragment.String()
		fileHashesByNs[nsConf.Name] = util.Hash("", renderedConfig)
		// don't validate the admin namespace, just render it
		err = util.WriteStringToFile(filepath.Join(outputDir, "admin-ns.conf"), renderedConfig)
		if err != nil {
			logrus.Infof("Cannot store config file for namespace %s", nsConf.Name)
		}

		break
	}

	for _, nsConf := range g.model {
		if nsConf.Name == g.cfg.AdminNamespace {
			continue
		}

		var renderedConfig, configHash string

		prepConfig, err := extractPrepConfig(nsConf.Name, prepareConfigs)

		if err == nil {
			// render config
			renderedConfig, _, err = g.makeNamespaceConfiguration(nsConf, genCtx, onlyProcess)
			configHash = util.Hash("", renderedConfig+prepConfig)
		}

		if err != nil {
			configHash = util.Hash("ERROR", err.Error())
			logrus.Infof("Configuration for namespace %s cannot be validated: %+v", nsConf.Name, err)
			if nsConf.PreviousConfigHash != configHash {
				g.updateStatus(nsConf.Name, err.Error())
			}
			fileHashesByNs[nsConf.Name] = configHash
			continue
		}

		// namespace is not configured
		if renderedConfig == "" {
			fileHashesByNs[nsConf.Name] = configHash
			if nsConf.PreviousConfigHash != configHash && nsConf.IsKnownFromBefore {
				// empty config is a valid input, clear error status
				g.updateStatus(nsConf.Name, "")
			}
			// If a config file had been created, remove it
			unusedFile := filepath.Join(outputDir, fmt.Sprintf("ns-%s.conf", nsConf.Name))
			err := os.Remove(unusedFile)
			if err != nil && !os.IsNotExist(err) {
				logrus.Warnf("Error removing unused file %s: %+v", unusedFile, err)
			}
			continue
		}

		var validationTrailer string

		if g.validator != nil {
			validationTrailer = g.makeValidationTrailer(nsConf, genCtx).String()
			err = g.validator.ValidateConfigExtremely(renderedConfig+"\n# validation  trailer:\n"+validationTrailer, nsConf.Name)

			if err != nil {
				logrus.Infof("Configuration for namespace %s cannot be validated with fluentd", nsConf.Name)
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
		model.PreprocessingDirectives = append(model.PreprocessingDirectives, prepConfig)
		fileHashesByNs[nsConf.Name] = configHash
		if g.cfg.FsDatasourceDir != "" {
			// if the source is the filesystem, preserve the validation trailer
			// so that generated files are valid in isolation
			renderedConfig = renderedConfig + "\n# validation  trailer:\n" + validationTrailer
		}
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

func (g *Generator) generatePrepareConfigs(genCtx *processors.GenerationContext) map[string]interface{} {
	prepareConfigs := map[string]interface{}{}
	for _, nsConf := range g.model {
		if nsConf.Name == g.cfg.AdminNamespace {
			continue
		}

		_, prep, err := g.makeNamespaceConfiguration(nsConf, genCtx, onlyPrepare)
		if err != nil {
			prepareConfigs[nsConf.Name] = err
		} else {
			prepareConfigs[nsConf.Name] = prep
		}
	}
	return prepareConfigs
}

func (g *Generator) makeValidationTrailer(ns *datasource.NamespaceConfig, genCtx *processors.GenerationContext) fluentd.Fragment {
	fragment, err := fluentd.ParseString(ns.FluentdConfig)
	if err != nil {
		return nil
	}

	ctx := g.makeContext(ns, genCtx)

	return processors.GetValidationTrailer(fragment, ctx, processors.DefaultProcessors()...)
}

func (g *Generator) makeContext(ns *datasource.NamespaceConfig, genCtx *processors.GenerationContext) *processors.ProcessorContext {
	ctx := &processors.ProcessorContext{
		Namepsace:         ns.Name,
		NamespaceLabels:   ns.Labels,
		AllowFile:         g.cfg.AllowFile,
		DeploymentID:      g.cfg.ID,
		MiniContainers:    ns.MiniContainers,
		KubeletRoot:       g.cfg.KubeletRoot,
		GenerationContext: genCtx,
		AllowTagExpansion: g.cfg.AllowTagExpansion,
	}
	return ctx
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
	model := struct {
		ID                string
		PrometheusEnabled bool
	}{
		ID:                util.MakeFluentdSafeName(g.cfg.ID),
		PrometheusEnabled: g.cfg.PrometheusEnabled,
	}

	buf := &bytes.Buffer{}
	err = tmpl.Execute(buf, model)
	if err != nil {
		logrus.Warnf("Error rendering template file %s: %+v", templateFile, err)
		return
	}

	util.WriteStringToFile(dest, buf.String())
}

// CleanupUnusedFiles removes "ns-*.conf" files of namespaces that are no more existent
func (g *Generator) CleanupUnusedFiles(outputDir string, namespaces map[string]string) {
	files, err := filepath.Glob(fmt.Sprintf("%s/ns-*.conf", outputDir))
	if err != nil {
		logrus.Warnf("Error finding unused files: %+v", err)
		return
	}

	for _, f := range files {
		ns := strings.TrimSuffix(strings.TrimPrefix(filepath.Base(f), "ns-"), ".conf")
		if _, ok := namespaces[ns]; !ok {
			if err := os.Remove(f); err != nil {
				logrus.Warnf("Error removing unused file %s: %+v", f, err)
			}
		}
	}
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
