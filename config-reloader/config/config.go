// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package config

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/alecthomas/kingpin"
	"github.com/sirupsen/logrus"
	"github.com/vmware/kube-fluentd-operator/config-reloader/util"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	// Version is the current version of the app, generated at build time
	Version = "unknown"
)

// Config is a project-wide configuration
type Config struct {
	Master                 string
	KubeConfig             string
	FluentdRPCPort         int
	TemplatesDir           string
	OutputDir              string
	LogLevel               string
	FluentdLogLevel        string
	BufferMountFolder      string
	AnnotConfigmapName     string
	AnnotStatus            string
	DefaultConfigmapName   string
	IntervalSeconds        int
	Datasource             string
	CRDMigrationMode       bool
	FsDatasourceDir        string
	AllowFile              bool
	ID                     string
	FluentdValidateCommand string
	MetaKey                string
	MetaValues             string
	LabelSelector          string
	KubeletRoot            string
	Namespaces             []string
	PrometheusEnabled      bool
	MetricsPort            int
	AllowTagExpansion      bool
	AdminNamespace         string
	// parsed or processed/cached fields
	level               logrus.Level
	ParsedMetaValues    map[string]string
	ParsedLabelSelector labels.Set
	ExecTimeoutSeconds  int
	ReadBytesLimit      int
}

var defaultConfig = &Config{
	Master:               "",
	KubeConfig:           "",
	FluentdRPCPort:       24444,
	TemplatesDir:         "/templates",
	OutputDir:            "/fluentd/etc",
	Datasource:           "default",
	LogLevel:             logrus.InfoLevel.String(),
	FluentdLogLevel:      "info",
	BufferMountFolder:    "",
	AnnotConfigmapName:   "logging.csp.vmware.com/fluentd-configmap",
	AnnotStatus:          "logging.csp.vmware.com/fluentd-status",
	DefaultConfigmapName: "fluentd-config",
	KubeletRoot:          "/var/lib/kubelet/",
	IntervalSeconds:      60,
	ID:                   "default",
	PrometheusEnabled:    false,
	MetricsPort:          9000,
	AdminNamespace:       "kube-system",
	ExecTimeoutSeconds:   30,
	ReadBytesLimit:       51200,
}

var reValidID = regexp.MustCompile("([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]")
var reValidAnnotationName = regexp.MustCompile("^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]+.*$")

func (cfg *Config) GetLogLevel() logrus.Level {
	return cfg.level
}

// Validate performs validation on the Config object
// nolint:gocognit
func (cfg *Config) Validate() error {
	if cfg.IntervalSeconds < 0 {
		// better normalize then fail
		cfg.IntervalSeconds = 60
	}

	if cfg.ExecTimeoutSeconds < 0 {
		cfg.ExecTimeoutSeconds = 30
	}

	ll, err := logrus.ParseLevel(cfg.LogLevel)
	if err != nil {
		return fmt.Errorf("failed to parse log level: %+v", err)
	}
	cfg.level = ll

	fll, err := cfg.ParseFluentdLogLevel()
	if err != nil {
		return fmt.Errorf("failed to parse fluentd log level: %+v", err)
	}
	cfg.FluentdLogLevel = fll

	if !reValidID.MatchString(cfg.ID) {
		return fmt.Errorf("ID must be a valid hostname")
	}

	if cfg.AnnotConfigmapName == "" || !reValidAnnotationName.MatchString(cfg.AnnotConfigmapName) {
		return fmt.Errorf("invalid annotation name: '%s'", cfg.AnnotConfigmapName)
	}

	if cfg.BufferMountFolder != "" && !cfg.hasValidBufferMountFolder() {
		return fmt.Errorf("invalid fluentd buffer mount folder: %v%v", "/var/log/", cfg.BufferMountFolder)
	}

	// this can be empty
	if cfg.AnnotStatus != "" && !reValidAnnotationName.MatchString(cfg.AnnotStatus) {
		return fmt.Errorf("invalid annotation name: '%s'", cfg.AnnotStatus)
	}

	if cfg.Datasource == "fs" && cfg.FsDatasourceDir == "" {
		return errors.New("using --datasource=fs requires --fs-dir too")
	}

	if cfg.MetaKey != "" && cfg.MetaValues == "" {
		return errors.New("using --meta-key requires --meta-values too")
	}
	isValid := func(s string) bool {
		if len(s) == 0 {
			return false
		}
		if strings.ContainsRune(s, '\'') {
			return false
		}
		return true
	}

	if cfg.MetaKey != "" {
		cfg.ParsedMetaValues = map[string]string{}
		values := strings.Split(cfg.MetaValues, ",")

		for _, ele := range values {
			if len(ele) == 0 {
				// trailing or double ,,
				continue
			}
			kvp := strings.Split(ele, "=")
			if len(kvp) != 2 {
				return fmt.Errorf("bad metadata: %s, use the k=v,k2=v2... format", cfg.MetaValues)
			}
			k := util.Trim(kvp[0])
			v := util.Trim(kvp[1])

			if isValid(k) && isValid(v) {
				cfg.ParsedMetaValues[k] = v
			}
		}

		if len(cfg.ParsedMetaValues) == 0 {
			return errors.New("using --meta-key requires --meta-values too")
		}
	}

	if cfg.Datasource == "multimap" {
		if cfg.LabelSelector == "" {
			return errors.New("using --datasource=multimap requires --label-selector too")
		}

		parsed := map[string]string{}
		values := strings.Split(cfg.LabelSelector, ",")

		for _, ele := range values {
			if len(ele) == 0 {
				// trailing or double ,,
				continue
			}
			kvp := strings.Split(ele, "=")
			if len(kvp) != 2 {
				return fmt.Errorf("bad label selector: %s, use the k=v,k2=v2... format", cfg.MetaValues)
			}
			k := util.Trim(kvp[0])
			v := util.Trim(kvp[1])

			if isValid(k) && isValid(v) {
				parsed[k] = v
			}
		}
		cfg.ParsedLabelSelector = labels.Set(parsed)
	}

	return nil
}

func (cfg *Config) ParseFlags(args []string) error {
	app := kingpin.New("config-reloader", "Regenerates Fluentd configs based Kubernetes namespace annotations against templates, reloading Fluentd if necessary")
	app.Version(Version)
	app.DefaultEnvars()

	// Flags related to Kubernetes
	app.Flag("master", "The Kubernetes API server to connect to (default: auto-detect)").Default(defaultConfig.Master).StringVar(&cfg.Master)
	app.Flag("kubeconfig", "Retrieve target cluster configuration from a Kubernetes configuration file (default: auto-detect)").Default(defaultConfig.KubeConfig).StringVar(&cfg.KubeConfig)

	app.Flag("datasource", "Datasource to use default|fake|fs|multimap|crd (default: default) ").Default("default").EnumVar(&cfg.Datasource, "default", "fake", "fs", "multimap", "crd")
	app.Flag("crd-migration-mode", "Enable the crd datasource together with the current datasource to facilitate the migration (used only with --datasource=default|multimap)").BoolVar(&cfg.CRDMigrationMode)
	app.Flag("fs-dir", "If --datasource=fs is used, configure the dir hosting the files").StringVar(&cfg.FsDatasourceDir)

	app.Flag("interval", "Run every x seconds").Default(strconv.Itoa(defaultConfig.IntervalSeconds)).IntVar(&cfg.IntervalSeconds)

	app.Flag("allow-file", "Allow @type file for namespace configuration").BoolVar(&cfg.AllowFile)

	app.Flag("id", "The id of this deployment. It is used internally so that two deployments don't overwrite each other's data").Default(defaultConfig.ID).StringVar(&cfg.ID)

	app.Flag("fluentd-rpc-port", "RPC port of Fluentd").Default(strconv.Itoa(defaultConfig.FluentdRPCPort)).IntVar(&cfg.FluentdRPCPort)
	app.Flag("log-level", "Control verbosity of log level for reloader").Default(defaultConfig.LogLevel).StringVar(&cfg.LogLevel)
	app.Flag("fluentd-loglevel", "Control verbosity of log level for fluentd").Default(defaultConfig.FluentdLogLevel).StringVar(&cfg.FluentdLogLevel)

	app.Flag("buffer-mount-folder", "Folder in /var/log/{} where to create all fluentd buffers").Default(defaultConfig.BufferMountFolder).StringVar(&cfg.BufferMountFolder)

	app.Flag("annotation", "Which annotation on the namespace stores the configmap name?").Default(defaultConfig.AnnotConfigmapName).StringVar(&cfg.AnnotConfigmapName)
	app.Flag("default-configmap", "Read the configmap by this name if namespace is not annotated. Use empty string to suppress the default.").Default(defaultConfig.DefaultConfigmapName).StringVar(&cfg.DefaultConfigmapName)
	app.Flag("status-annotation", "Store configuration errors in this annotation, leave empty to turn off").Default(defaultConfig.AnnotStatus).StringVar(&cfg.AnnotStatus)

	app.Flag("prometheus-enabled", "Prometheus metrics enabled (default: false)").BoolVar(&cfg.PrometheusEnabled)
	app.Flag("metrics-port", "Expose prometheus metrics on this port (also needs --prometheus-enabled)").Default(strconv.Itoa(defaultConfig.MetricsPort)).IntVar(&cfg.MetricsPort)

	app.Flag("kubelet-root", "Kubelet root dir, configured using --root-dir on the kubelet service").Default(defaultConfig.KubeletRoot).StringVar(&cfg.KubeletRoot)
	app.Flag("namespaces", "List of namespaces to process. If empty, processes all namespaces").StringsVar(&cfg.Namespaces)

	app.Flag("templates-dir", "Where to find templates").Default(defaultConfig.TemplatesDir).StringVar(&cfg.TemplatesDir)
	app.Flag("output-dir", "Where to output config files").Default(defaultConfig.OutputDir).StringVar(&cfg.OutputDir)

	app.Flag("meta-key", "Attach metadata under this key").StringVar(&cfg.MetaKey)
	app.Flag("meta-values", "Metadata in the k=v,k2=v2 format").StringVar(&cfg.MetaValues)

	app.Flag("fluentd-binary", "Path to fluentd binary used to validate configuration").StringVar(&cfg.FluentdValidateCommand)

	app.Flag("label-selector", "Label selector in the k=v,k2=v2 format (used only with --datasource=multimap)").StringVar(&cfg.LabelSelector)

	app.Flag("allow-tag-expansion", "Allow specifying tags in the format 'k.{a,b}.** k.c.**' (default: false)").BoolVar(&cfg.AllowTagExpansion)

	app.Flag("admin-namespace", "Configurations defined in this namespace are copied as is, without further processing. Virtual plugins can also be defined in this namespace").Default(defaultConfig.AdminNamespace).StringVar(&cfg.AdminNamespace)

	app.Flag("exec-timeout", "Timeout duration (in seconds) for exec command during validation").Default(strconv.Itoa(defaultConfig.ExecTimeoutSeconds)).IntVar(&cfg.ExecTimeoutSeconds)

	app.Flag("container-bytes-limit", "read_bytes_limit_per_second parameter for tail plugin per container file. Default 2MB/min").Default(strconv.Itoa(defaultConfig.ReadBytesLimit)).IntVar(&cfg.ReadBytesLimit)

	_, err := app.Parse(args)

	if err != nil {
		return err
	}

	return nil
}

// ParseFluentdLogLevel takes a string level and returns the Fluentd log level constant.
func (cfg *Config) ParseFluentdLogLevel() (string, error) {
	switch strings.ToLower(cfg.FluentdLogLevel) {
	case "fatal":
		return "fatal", nil
	case "error":
		return "error", nil
	case "warn", "warning":
		return "warn", nil
	case "info":
		return "info", nil
	case "debug":
		return "debug", nil
	case "trace":
		return "trace", nil
	}

	return "", fmt.Errorf("not a valid Fluentd log Level: %q", cfg.FluentdLogLevel)
}

// isValidFolderName takes a folder name and checks if there are any dangerous characters:
func (cfg *Config) hasValidBufferMountFolder() bool {
	for _, r := range cfg.BufferMountFolder {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' {
			return false
		}
		continue
	}
	return true
}
