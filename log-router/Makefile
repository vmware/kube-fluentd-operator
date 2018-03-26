# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

HELM_PACKAGE := log-router

.DEFAULT_GOAL := helm-package
-include vmware.mk

helm-package: clean
	helm package .

helm-dry-run:
	helm install . --dry-run --debug \
	  --set image.pullSecret=test-pull-secret \
	  --set meta.key=key \
	  --set meta-values=env=test

clean:
	rm -fr $(HELM_PACKAGE)-*.tgz || true
