// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package processors

import (
	"fmt"
	"testing"

	"github.com/vmware/kube-fluentd-operator/config-reloader/fluentd"

	"github.com/stretchr/testify/assert"
)

func Test_callForEveryDirective(t *testing.T) {
	var s = `
<match **>
    @type logzio
	<buffer>
		@type file
		path /etc/passwd
		<nested>
		</nested>
  </buffer>
</match>

<match **>
  @type logzio
	<buffer>
		@type file
		path /etc/passwd
  </buffer>
</match>
`
	fragment, err := fluentd.ParseString(s)
	assert.Nil(t, err)

	fmt.Printf("Original:\n%s", fragment)

	ctx := &ProcessorContext{
		Namepsace: "test",
	}
	count := 0
	inc := func(*fluentd.Directive, *ProcessorContext) error {
		count++
		return nil
	}

	err = applyRecursivelyInPlace(fragment, ctx, inc)
	assert.Nil(t, err, "was error instead %+v", err)
	assert.Equal(t, 5, count)
}
