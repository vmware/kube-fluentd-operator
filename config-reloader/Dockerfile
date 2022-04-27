# Copyright © 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

# builder image
FROM golang:1.17.9 as builder

WORKDIR /go/src/github.com/vmware/kube-fluentd-operator/config-reloader
COPY . .

# Speed up local builds where vendor is populated
ARG VERSION
# Good to have another test run as this container is different than lint container
RUN make in-docker-test
RUN make build VERSION=$VERSION

# always use the un-pushable image
FROM vmware/base-fluentd-operator:latest

COPY templates /templates
COPY validate-from-dir.sh /bin/validate-from-dir.sh
COPY --from=builder /go/src/github.com/vmware/kube-fluentd-operator/config-reloader/config-reloader /bin/config-reloader
