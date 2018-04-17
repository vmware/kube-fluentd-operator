package processors

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"
)

func TestMountedFileRemovedAfterProcessing(t *testing.T) {
	s := `
<source>
  @type mounted-file
  path /hello/world
  labels app=spring-mvc  
</source>

<match **>
  @type logzio
</match>
	`

	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original: %s", fragment)

	ctx := &ProcessorContext{
		Namepsace: "monitoring",
	}

	processor := &MountedFileState{}
	fragment, err = Apply(fragment, ctx, processor)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(fragment))

	assert.Equal(t, 1, len(processor.ContainerFiles))

	cf := processor.ContainerFiles[0]
	assert.Equal(t, "/hello/world", cf.Path)
	assert.Equal(t, map[string]string{"app": "spring-mvc"}, cf.Labels)
}

func TestMountedFileCatchesMissingFile(t *testing.T) {
	missingPath := `
	<source>
	  @type mounted-file
	  labels app=spring-mvc  
	</source>
	
	<match **>
	  @type logzio
	</match
	`
	fragment, err := fluentd.ParseString(missingPath)
	assert.Nil(t, fragment)
	assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
}

func TestMountedFileCatchesEmptyLabels(t *testing.T) {
	missingPath := `
	<source>
	  @type mounted-file
	  labels 
	</source>
	
	<match **>
	  @type logzio
	</match
	`
	fragment, err := fluentd.ParseString(missingPath)
	assert.Nil(t, fragment)
	assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
}

func TestMountedFileCatchesMissingLabels(t *testing.T) {
	missingPath := `
	<source>
	  @type mounted-file
	  file /etc/hosts
	</source>
	
	<match **>
	  @type logzio
	</match
	`
	fragment, err := fluentd.ParseString(missingPath)
	assert.Nil(t, fragment)
	assert.NotNil(t, err, "Must have failed, instead parsed to %+v", fragment)
}
