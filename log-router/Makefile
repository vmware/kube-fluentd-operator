# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

HELM_REPO := https://csp-artifactory.eng.vmware.com/artifactory/list/csp-helm/
HELM_PACKAGE := fluentd-router

helm-package: clean
	helm package .

helm-publish: helm-package
	../../bin/upload-charts.sh --url $(HELM_REPO) $(HELM_PACKAGE)-*.tgz

helm-dry-run:
	helm install . --dry-run --debug \
	  --set image.pullSecret=test-pull-secret \
	  --set meta.key=key \
	  --set meta-values=env=test

clean:
	rm -fr $(HELM_PACKAGE)-*.tgz || true
