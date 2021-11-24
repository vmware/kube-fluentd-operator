// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package datasource

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/vmware/kube-fluentd-operator/config-reloader/util"

	"github.com/sirupsen/logrus"
)

type fsDatasource struct {
	confHashes      map[string]string
	rootDir         string
	statusOutputDir string
}

func (d *fsDatasource) GetNamespaces(ctx context.Context) ([]*NamespaceConfig, error) {
	res := []*NamespaceConfig{}

	files, err := filepath.Glob(fmt.Sprintf("%s/*.conf", d.rootDir))
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		base := filepath.Base(f)
		ns := base[0 : len(base)-5]
		contents, err := ioutil.ReadFile(f)
		if err != nil {
			logrus.Infof("Cannot read file %s: %+v", f, err)
			continue
		}

		cfg := &NamespaceConfig{
			Name:               ns,
			FluentdConfig:      string(contents),
			PreviousConfigHash: d.confHashes[ns],
		}

		logrus.Infof("Loading namespace %s from file %s", ns, f)
		res = append(res, cfg)
	}

	return res, nil
}

func (d *fsDatasource) WriteCurrentConfigHash(namespace string, hash string) {
	d.confHashes[namespace] = hash
}

func (d *fsDatasource) UpdateStatus(ctx context.Context, namespace string, status string) {
	fname := filepath.Join(d.statusOutputDir, fmt.Sprintf("ns-%s.status", namespace))
	if status != "" {
		util.WriteStringToFile(fname, status)
	} else {
		os.Remove(fname)
	}
}

// NewFileSystemDatasource turns all files matching *.conf patter in the given dir into namespace configs
func NewFileSystemDatasource(ctx context.Context, rootDir string, statusOutputDir string) Datasource {
	return &fsDatasource{
		confHashes:      make(map[string]string),
		rootDir:         rootDir,
		statusOutputDir: statusOutputDir,
	}
}
