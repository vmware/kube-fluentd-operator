# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

IMAGE            ?= vmware/kube-fluentd-operator
TAG              ?= latest

VERSION          ?= $(shell git describe --tags --always --dirty)
BUILD_FLAGS      := -v
LDFLAGS          := -X github.com/vmware/kube-fluentd-operator/config-reloader/config.Version=$(VERSION) -w -s

SHELL             = bash
PKG               = vmware/kube-fluentd-operator

GO_OPTS           = GO111MODULE=on GOARCH=amd64 CGO_ENABLED=0 GOBIN=$(GOBIN)
GO                = $(GO_OPTS) go

CURRENT_DIR       = $(shell pwd)
DEV_ENV_IMAGE    := vmwaresaas.jfrog.io/vdp-public/go-dev:latest
DEV_ENV_WORK_DIR := /go/src/${PKG}
DEV_ENV_CMD      := docker run --rm -v ${CURRENT_DIR}:${DEV_ENV_WORK_DIR} -w ${DEV_ENV_WORK_DIR} ${DEV_ENV_IMAGE}

.DEFAULT_GOAL    := build-image

all: test build

dev:
	${DEV_ENV_CMD} bash

vendor:
	${GO} mod vendor

install:
	${GO} install -v .

tidy:
	${GO} mod tidy -v

test: lint test-unit

lint: vendor
	${DEV_ENV_CMD} lint

test-unit: vendor
	${DEV_ENV_CMD} go test --cover --race -v ./...

in-docker-test:
	go test --cover --race -v ./...

test-cover: vendor
	${DEV_ENV_CMD} test-cover.sh

report-cover: test-cover
	${DEV_ENV_CMD} report-cover.sh

test-clean:
	rm -rf coverage.*

build:
	${GO} build $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

dep:
	which dep > /dev/null || (echo "Install dep first: go get -u github.com/golang/dep/cmd/dep" && exit 1)
	dep ensure

guess-tag:
	@echo "TAG=`git describe --tags --always`"

clean:
	rm -fr config-reloader pkg > /dev/null

build-image:
	docker build --build-arg VERSION=$(VERSION) -t $(IMAGE):$(TAG) .

push-image: build-image
	docker push $(IMAGE):$(TAG)

push-latest:
	docker tag $(IMAGE):$(TAG) $(IMAGE):latest
	docker push $(IMAGE):latest

create-test-ns:
	HUMIO_KEY=$(HUMIO_KEY) LOGGLY_TOKEN=$(LOGGLY_TOKEN) envsubst '$$LOGGLY_TOKEN:$$HUMIO_KEY' < examples/manifests/kfo-test.yaml | kubectl apply -f -

delete-test-ns:
	kubectl delete -f examples/manifests/kfo-test.yaml

run-loop-fs: install
	rm -fr tmp
	./config-reloader \
		--interval 5 \
		--log-level=debug \
		--output-dir=tmp \
		--meta-key=prefix \
		--meta-values=aws_region=us-west-2,csp_cluster=mon \
		--templates-dir=templates \
		--datasource=fs \
		--fs-dir=examples \
		--fluentd-binary "fluentd/fake-fluentd.sh -p /plugins"

run-once-fs: install
	rm -fr tmp
	./config-reloader \
		--interval 0 \
		--log-level=debug \
		--fluentd-loglevel=debug \
		--buffer-mount-folder="" \
		--output-dir=tmp \
		--meta-key=prefix2 \
		--meta-values=aws_region=us-west-2,csp_cluster=mon \
		--templates-dir=templates \
		--datasource=fs \
		--fs-dir=examples \
		--fluentd-binary "fluentd/fake-fluentd.sh -p /plugins"

run-once: install
	rm -fr tmp
	./config-reloader \
		--interval 0 \
		--log-level=debug \
		--fluentd-loglevel=debug \
		--buffer-mount-folder="" \
		--output-dir=tmp \
		--templates-dir=templates \
		--meta-key=run-once \
		--meta-values=aws_region=us-west-2,csp_cluster=mon \
		--fluentd-binary "fluentd/fake-fluentd.sh -p /plugins"

run-loop: install
	rm -fr tmp
	./config-reloader \
		--interval 5 \
		--log-level=debug \
		--fluentd-loglevel=debug \
		--buffer-mount-folder="" \
		--output-dir=tmp \
		--meta-key=prefix3 \
		--meta-values=aws_region=us-west-2,csp_cluster=mon \
		--templates-dir=templates

run-fluentd:
	docker run --entrypoint=fluentd \
		-ti --rm -v `pwd`:/workspace --net=host \
		$(IMAGE):$(TAG) \
		-p /fluentd/plugins -v -c /workspace/local-fluent.conf

dep-graph:
	godepgraph -s \
		-l 4 \
		-p github.com/alecthomas/units,github.com/alecthomas/template,github.com/spf13,github.com/jackc,k8s.io/kubernetes,k8s.io/apimachinery,github.com/palantir,github.com/sirupsen,github.com/prometheus,golang.org,gopkg.in \
		github.com/vmware/kube-fluentd-operator/config-reloader \
		| sed 's|github.com/vmware/kube-fluentd-operator/config-reloader/||g'\
		| dot -Tpng -o godepgraph.png

validate-config:
	docker run --entrypoint=fluentd \
		-ti --rm -v `pwd`:/workspace --net=host \
		$(IMAGE):$(TAG) \
		--dry-run -p /fluentd/plugins -v -c /workspace/tmp/fluent.conf

shell:
	docker run --entrypoint=/bin/bash \
		-ti --rm -v `pwd`:/workspace --net=host \
		$(IMAGE):$(TAG)
