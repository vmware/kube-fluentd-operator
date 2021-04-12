# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

HELM_PACKAGE := log-router

.DEFAULT_GOAL := helm-package

helm-package: clean
	helm lint .
	helm package .

helm-dry-run:
	helm install . --dry-run --debug \
	  --set image.pullSecret=test-pull-secret \
	  --set namespaces[0]=kfo-test \
	  --set namespaces[1]=kfo-consumer \
	  --set rbac.create=true \
	  --set meta.key=key \
	  --set meta.values.env=test \
	  --set meta.values.region=eu \
	  --set tolerations[0].key=key \
	  --set tolerations[0].operator=op \
	  --set tolerations[0].value=value \
	  --set extraEnv.HELLO=world \
	  --set updateStrategy.type=OnDelete

clean:
	rm -fr $(HELM_PACKAGE)-*.tgz || true
