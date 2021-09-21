# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

IMAGE            ?= vmware/base-fluentd-operator
IMG_ARCH         ?= amd64
WIN_TAG          ?=
IMG_OS           ?= linux
TAG              ?= $(IMG_ARCH)-$(IMG_OS)$(shell v='$(WIN_TAG)'; echo "$${v:+-}$${v:-}")

.PHONY: test
ttt:
	echo $(TAG)

build-image:
	DOCKER_BUILDKIT=1 docker build -t $(IMAGE):$(TAG) .

build-windows-image:
	@echo Building $(IMAGE):$(TAG)
	docker build --network "Default Switch" -f Dockerfile.windows --build-arg WIN_TAG=$(WIN_TAG) -t $(IMAGE):$(TAG) .


shell:
	docker run --entrypoint=/bin/bash \
	  -ti --rm -v `pwd`:/workspace --net=host \
	  $(IMAGE):$(TAG)

run: build-image
	docker run -ti --rm \
	  --net=host \
	  -v `pwd`/failsafe.conf:/fluentd/etc/fluent.conf \
	  -v /var/log:/var/log \
	  -v /var/run:/var/run \
	  -e FLUENTD_OPT="--no-supervisor" \
	  $(IMAGE):$(TAG)

test: build-image
	docker run -ti --rm \
	  --net=host \
	  -v `pwd`:/workspace \
	  -v `pwd/plugins`:/fluentd/plugins \
	  -v `pwd`/test/local.conf:/fluentd/etc/fluent.conf \
	  -e FLUENTD_OPT="--no-supervisor" \
		-e PAPERTRAIL_PORT=$$PAPERTRAIL_PORT \
		-e PAPERTRAIL_HOST=$$PAPERTRAIL_HOST \
	  $(IMAGE):$(TAG)

list-gems:
	@docker run -ti --rm \
	  --net=host --entrypoint /bin/bash \
	  $(IMAGE):$(TAG) \
	  -c 'fluent-gem list' | \
	  grep '^fluent' | sed 's/^/* /'
