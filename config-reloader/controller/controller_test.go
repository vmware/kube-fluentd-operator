package controller

import (
	"context"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/config"
	"github.com/vmware/kube-fluentd-operator/config-reloader/datasource"
)

// UnitTest for RunOnce
func TestRunOnceController(t *testing.T) {
	// 1. Create new controller
	// 2. RunOnce controller
	// 3. Create a new namespace in the folder
	// 4. runOnce controller
	assert := assert.New(t)
	config := config.Config{
		Datasource:      "fs",
		FsDatasourceDir: "../examples",
		TemplatesDir:    "../templates",
		ID:              "default",
		OutputDir:       "../tmp",
		LogLevel:        "debug",
	}
	expectedResult := 3
	// Prepare TestCase
	ctx := context.Background()
	ds := datasource.NewFileSystemDatasource(ctx, config.FsDatasourceDir, config.OutputDir)

	// Create controller
	up := NewFixedTimeUpdater(ctx, config.IntervalSeconds)
	// 1. Create new controller
	ctrl, err := New(ctx, &config, ds, up)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	// 2. RunOnce controller
	err = ctrl.RunOnce(ctx)
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	assert.Equal(expectedResult, ctrl.GetTotalConfigNS())

	// 3. Create a new namespace in the folder
	newNamespaceFile := config.FsDatasourceDir + "/new-namespace.conf"
	configData := []byte("<match **>\n@type stdout\n</match>")
	defer os.Remove(newNamespaceFile)
	err = os.WriteFile(newNamespaceFile, configData, 0644)
	if err != nil {
		logrus.Fatalf(err.Error())
	}

	// 4. RunOnce controller
	err = ctrl.RunOnce(ctx)
	if err != nil {
		logrus.Fatalf(err.Error())
	}
	assert.Equal(expectedResult+1, ctrl.GetTotalConfigNS())
}
