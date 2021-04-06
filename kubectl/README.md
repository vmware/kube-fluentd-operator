### Installing log-router manually with kubectl:

We support installing log-router (KFO) with Helm 3. However, in cases where that is not possible or not preferred, these instructions should suffice in getting started with log-router in a k8s cluster. You are responsible for figuring out which [flags](https://github.com/vmware/kube-fluentd-operator#synopsis) and environment variables need to be set on the log-router containers for your particular environment.

#### Install the CRD:
```bash
kubectl apply -f ./crd.yaml
```


#### Install the other kubectl manifests:

```bash
# Get the latest tag from github via curl:
export kube_latest_tag=$(curl --silent "https://api.github.com/repos/vmware/kube-fluentd-operator/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

# Inject the latest tag into the manifests:
sed -i .bak -e "s|<INSERT_TAG_HERE>|${kube_latest_tag}|g" manifests.yaml

# Install with kubectl:
kubectl apply -f ./manifests.yaml
```
