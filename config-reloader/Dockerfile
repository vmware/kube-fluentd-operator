# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

# builder image
FROM golang:1.13 as builder

WORKDIR /go/src/github.com/vmware/kube-fluentd-operator/config-reloader
COPY . .

# Speed up local builds where vendor is populated
ARG VERSION
RUN make test
RUN make build VERSION=$VERSION

# always use the unpushable image
FROM vmware/base-fluentd-operator:latest

COPY templates /templates
COPY validate-from-dir.sh /bin/validate-from-dir.sh
COPY --from=builder /go/src/github.com/vmware/kube-fluentd-operator/config-reloader/config-reloader /bin/config-reloader
