// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type Mount struct {
	Path       string
	VolumeName string
	SubPath    string
}

// MiniContainer container subset with the parent pod's metadata
type MiniContainer struct {
	// the pod id
	PodID   string
	PodName string

	Image       string
	ContainerID string

	// pod labels
	Labels map[string]string

	// container name
	Name string
	// only the emptyDir mounts, never empty, sorted by len(Path), descending
	HostMounts []*Mount

	NodeName string
}

func TestMakeFluentdSafeName(t *testing.T) {
	assert.Equal(t, "a", MakeFluentdSafeName("a"))
	assert.Equal(t, "123", MakeFluentdSafeName("123"))
	assert.Equal(t, "", MakeFluentdSafeName(""))
	assert.Equal(t, "a-a", MakeFluentdSafeName("a.a"))
	assert.Equal(t, "a-a", MakeFluentdSafeName("a\na"))
	assert.Equal(t, "---", MakeFluentdSafeName("   "))
}

func TestToRubyMapLiteral(t *testing.T) {
	assert.Equal(t, "{}", ToRubyMapLiteral(map[string]string{}))
	assert.Equal(t, "{'a'=>'1'}", ToRubyMapLiteral(map[string]string{
		"a": "1",
	}))
	assert.Equal(t, "{'a'=>'1','z'=>'2'}", ToRubyMapLiteral(map[string]string{
		"a": "1",
		"z": "2",
	}))
}

func TestTrim(t *testing.T) {
	assert.Equal(t, "a", Trim("a"))
	assert.Equal(t, "a", Trim("  a"))
	assert.Equal(t, "a", Trim("a  \t "))
	assert.Equal(t, "a", Trim(" \t a   "))
}

func TestTrimTrailingComment(t *testing.T) {
	assert.Equal(t, "a", TrimTrailingComment("a #12451345"))
	assert.Equal(t, "a", TrimTrailingComment("a"))
	assert.Equal(t, "a", TrimTrailingComment("a#########"))
}

func TestMakeStructureHash(t *testing.T) {
	mini1 := &MiniContainer{
		PodID:       "4b519aaf-67f1-4588-8164-f679b2298e25",
		PodName:     "kfo-log-router-nwxtj",
		Name:        "config-reloader",
		NodeName:    "vdp-dev-control-plane",
		Image:       "testing/kfo:delete-problems-3",
		ContainerID: "containerd://37dce75ed2f01c5f858b4c4cc96b23ebacaba6569af93ed64b3904be9a676cb1",
	}

	hashMini1, err := MakeStructureHash(mini1)
	assert.Nil(t, err)
	assert.Equal(t, uint64(0xa92a93a3863f8fd6), hashMini1)
}

func TestAreStructureHashEqual(t *testing.T) {
	mini1 := &MiniContainer{
		PodID:       "4b519aaf-67f1-4588-8164-f679b2298e25",
		PodName:     "kfo-log-router-nwxtj",
		Name:        "config-reloader",
		NodeName:    "vdp-dev-control-plane",
		Image:       "testing/kfo:delete-problems-3",
		ContainerID: "containerd://37dce75ed2f01c5f858b4c4cc96b23ebacaba6569af93ed64b3904be9a676cb1",
	}
	mini2 := &MiniContainer{
		PodID:       "4b519aaf-67f1-4588-8164-f679b2298e25",
		PodName:     "kfo-log-router-nwxtj",
		Name:        "config-reloader",
		NodeName:    "vdp-dev-control-plane",
		Image:       "testing/kfo:delete-problems-3",
		ContainerID: "containerd://37dce75ed2f01c5f858b4c4cc96b23ebacaba6569af93ed64b3904be9a676cb1",
	}
	mini3 := &MiniContainer{
		PodID:       "4b519aaf-67f1-4588-8164-f679b2298e25",
		PodName:     "kfo-log-router-next",
		Name:        "config-reloader",
		NodeName:    "vdp-dev-control-plane",
		Image:       "testing/kfo:delete-problems-3",
		ContainerID: "containerd://37dce75ed2f01c5f858b4c4cc96b23ebacaba6569af93ed64b3904be9a676cb1",
	}

	assert.Equal(t, true, AreStructureHashEqual(mini1, mini2))
	assert.NotEqual(t, true, AreStructureHashEqual(mini1, mini3))
	assert.Equal(t, false, AreStructureHashEqual(mini1, mini3))
}
