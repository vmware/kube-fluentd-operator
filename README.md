# kube-fluentd-operator  [![Build Status](https://travis-ci.org/vmware/kube-fluentd-operator.svg?branch=master)](https://travis-ci.org/vmware/kube-fluentd-operator)

[![Go Report Card](https://goreportcard.com/badge/github.com/vmware/kube-fluentd-operator)](https://goreportcard.com/report/github.com/vmware/kube-fluentd-operator)

## Overview

*kube-fluentd-operator* configures Fluentd in a Kubernetes environment. It compiles a Fluentd configuration from configmaps (one per namespace) - similar to how Ingress controllers compiles Nginx configuration based on Ingress resources. This way only one instance of Fluentd can handle all log shipping while the cluster admin need not coordinate with namespace admins.

Cluster administrators set up Fluentd once only and namespace owners can configure log routing as they wish. *kube-fluentd-operator* will re-configure Fluentd accordingly and make sure logs originating from a namespace will not be accessible by other tenants/namespace.

*kube-fluentd-operator* also extends the Fluentd configuration language to make it possible to refer to pods based on their labels and the container. Thie enables for very fined-grained targeting of logs for the purpose of pre-processing before shipping.

## Try it out

The easiest way to get started is using the helm chart. As it is not published you need to check out the code:

```bash
git clone git@github.com:vmware/kube-fluentd-operator.git
helm install --name kfo ./kube-fluentd-operator/log-router --set rbac.create=true
```

Then create a namespace "demo" and a configmap describing where all logs from `demo` should go to. Finally, point the kube-fluentd-operator to the correct configmap using annotations.

```bash
kubectl create ns demo

# the annotation logging.csp.vmware.com/fluentd-configmap stores the name of the configmap
kubectl annotate demo logging.csp.vmware.com/fluentd-configmap=logging-config

cat > fluent.conf << EOF
<match **>
  @type null
</match>
EOF

# "logging-config" is the name stored in the annotation
kubectl create configmap logging-config --namespace demo --from-file=fluent.conf=fluent.conf
```

In a minute, this config map would be translated to something like this:

```xml
<match demo.**>
  @type null
</match>
```

Even though the tag `**` was used in the `<match>` directive, the kube-flunetd-operator correctly expands this to `demo.**`. Indeed, if another tag which does not start with `demo.` was used, it would have failed validation. Namespace admins can safely assume that they has a dedicated Fluentd for themselves.

All configuration errors are stored in the annotation `logging.csp.vmware.com/fluentd-status`. Try replacing `**` with an invalid tag like 'hello-world'. After a minute, verify that the error message looks like this:

```bash
# extract just the value of logging.csp.vmware.com/fluentd-status
kubectl get ns demo -o jsonpath='{.metadata.annotations.logging\.csp\.vmware\.com/fluentd-status}'
bad tag for <match>: hello-world. Tag must start with **, $thins or demo
```

When the configuration is valid again the fluentd-status is set to "".

To see kube-fluentd-operator in action you need a cloud log collector like logz.io, loggly, papertrail or ELK accessible from the K8S cluster. A simple loggly configuration looks like this (replace TOKEN with your customer token):

```xml
<match **>
   @type loggly
   loggly_url https://logs-01.loggly.com/inputs/TOKEN/tag/fluentd
</match>
```

## Build

Get the code using `go get`:

```bash
go get -u github.com/vmware/kube-fluentd-operator/config-reloader
cd $GOPATH/src/github.com/vmware/kube-fluentd-operator

# build a base-image
cd base-image && make

# build helm chart
cd log-router && make

# build the daemon
cd config-reloader
# get vendor/ deps, you need to have "dep" tool installed
make dep
make install

# run with mock data (loaded from the examples/ folder)
make run-local-fs

# inspect what is generated from the above command
ls -l tmp/
```

## Documentation

### Configuration language

To give the illusion that every namespace runs a dedicated Fluentd the user-provided configuration is post-processed. In general, expressions starting with `$` are macros that are expanded. These two directives are equivalent: `<match **>`, `<match $thins>`. Almost always, using the `**` is the preferred way to match logs: this way you can reuse the same configuration for multiple namespaces.

The `kube-system` is treated differently. Its configuration is not processed further as it is assumed only the cluster admin can manipulate resources in this namespace. If you don't plan to use advanced features described bellow, it is possible to route all logs from all namespaces using this configuration at the `kube-system` level:

```xml
<match **>
 @type ...
 # destination configuration omitted
</match>
```

`**` in this context is not processed and it means *literally* everything.

Fluentd assumes it is running in a distro with systemd and generates logs with these Fluentd tags:

* `systemd.{unit}`: the journal of a systemd unit, for example `systemd.docker.service`
* `docker`: all docker logs, not containers. If systemd is used, the docker logs are in `systemd.docker.service`
* `k8s.{component}`: logs from a K8S component, for example `k8s.kube-apiserver`
* `kube.{namespace}.{podid}.{container_name}`: a log originating from (namespace, pod, container)

As the `kube-system` is processed first, a match-all directive would consume all logs and any other namespace configuration will become irrelevant (unless `<copy>` is used).
A recommended configuration for the `kube-system` namespace is this one - it captures all but the user namespaces' logs:

```xml
<match systemd.** kube.kube-system.** k8s.** docker>
  # all k8s-internal and OS-level logs

  # destionation config omitted...
</match>
```

A very useful feature is the `<filter>` and the `$labels` macro to define parsing at the namespace level. For example, the config-reloader container uses the logfmt format. This makes it easy to use structured logging and ingest json data into a remote log ingestion service.

```xml
<filter $labels(app=log-router, _container=reloader)>
  @type parse
  format logfmt
  reserve_data true
</filter>

<match **>
  @type loggly
  # destionation config omitted
</match>
```

The above config will pipe all logs from the pods labelled with `app=log-router` through a [logfmt](https://github.com/vmware/kube-fluentd-operator/blob/master/base-image/plugins/parser_logfmt.rb) parser before sending them to loggly. Again, this configuration is valid in any namespace. If the namespace doesn't contain any `log-router` components then the `<filter>` directive is never activated. The `_container` is sort of a "meta" label and it allow for targeting the log stream of a specific container in a multi-container pod.

All plugins that change the fluentd tag are disabled for security reasons. Otherwise a rogue configuration may divert other namespace's logs to itself by prepending its name to the tag.

The `@type file` plugin is also disabled as it doesn't make sense to convert local docker json logs to another file on the same disk.

### Available plugins

`kube-fluentd-operator` aims to be easy to use and flexible. It also favors appending logs to multiple destinations using `<copy>` and as such comes with many destination plugins pre-installed:

* fluent-plugin-elasticsearch
* fluent-plugin-google-cloud
* fluent-plugin-logentries
* fluent-plugin-loggly
* fluent-plugin-logzio
* fluent-plugin-papertrail
* fluent-plugin-redis
* fluent-plugin-remote_syslog
* fluent-plugin-s3
* fluent-plugin-secure-forward

If you need other destination plugins you are welcome to contribute a patch or just create an issue.

### Log metadata

Often you run mulitple Kubernetes clusters but you need to aggregate all logs to a single destination. To be able to distinguish between different sources, `kube-fluentd-operator` can attach arbitrary metadata to every log event.
The metadata is nested under key chosen with `--meta-key`. Using the helm chart, metadata can be enbled like this:

```bash
helm instal ... --set=meta.key=metadata --set=meta.values=region=us-east-1\,env=staging\,cluster=legacy
```

Every log event, be it from a pod or a systemd unit, will now have carry this metadata:

```json
{
  "metadata": {
    "region": "us-east-1",
    "env": "staging",
    "cluster": "legacy",
  },
  "message": "hello world",
  "@timestamp": "..."
}
```

### Synopsis

The config-reloader binary is the one that listens to changes in K8S and generates Fluentd files. It runs as a daemonset and is not intended to interact with directly. The synopsis is useful when trying to understand the Helm chart or jsut hacking.

```bash
usage: config-reloader [<flags>]

Regenerates Fluentd configs based Kubernetes namespace annotations against templates, reloading Fluentd if necessary

Flags:
  --help                        Show context-sensitive help (also try --help-long and --help-man).
  --version                     Show application version.
  --master=""                   The Kubernetes API server to connect to (default: auto-detect)
  --kubeconfig=""               Retrieve target cluster configuration from a Kubernetes configuration file (default:
                                auto-detect)
  --datasource=default          Datasource to use
  --fs-dir=FS-DIR               If datasource=fs is used, configure the dir hosting the files
  --interval=60                 Run every x seconds
  --allow-file                  Allow @type file for namespace configuration
  --id="default"                The id of this deployment. It is used internally so that two deployments don't overwrite each
                                other's data
  --fluentd-rpc-port=24444      RPC port of Fluentd
  --log-level="info"            Control verbosity of log
  --annotation="logging.csp.vmware.com/fluentd-configmap"
                                Which annotation on the namespace stores the configmap name?
  --status-annotation="logging.csp.vmware.com/fluentd-status"
                                Store configuration errors in this annotation, leave empty to turn off
  --templates-dir="/templates"  Where to find templates
  --output-dir="/fluentd/etc"   Where to output config files
  --meta-key=META-KEY           Attach metadat under this key
  --meta-values=META-VALUES     Metadata in the k=v,k2=v2 format
  --fluentd-binary=FLUENTD-BINARY
                                Path to fluentd binary used to validate configuration

```

### Helm chart

| Parameter                                | Description                         | Default                                           |
|------------------------------------------|-------------------------------------|---------------------------------------------------|
| `rbac.create`                            | Create a serviceaccount+role, use if K8s is using RBAC        | `false`                  |
| `serviceAccountName`                     | Reuse an existing service account                | `""`                                             |
| `image.repositiry`                       | Repository                 | `jvassev/kube-fluentd-operator`                              |
| `image.tag`                              | Image tag                | `latest`                          |
| `image.pullPolicy`                       | Pull policy                 | `Always`                             |
| `image.pullSecret`                       | Optional pull secret name                 | `""`                                |
| `logLevel`                               | Default log level                 | `info`                               |
| `interval`                               | How often to check for config changes (seconds)                 | `45`          |
| `meta.key`                               | The metadata key (optional)                 | `""`                                |
| `meta.values`                            | Comma-separated key-values string (k1=v1,k2=v2,..)                 | Metadata to use for the key   |
| `fluentdResources`                       | Resource definitions for the fluentd container                 | `{}`|
| `reloaderResources`                      | Resource definitions for the reloader container              | `{}`                     |

## Cookbook

### I want to use one destination for everything

Simple, define configuration only for the kube-system namespace:

```bash
kube-system.conf:
<match **>
  # configure destination here
</match>
```

### I dont't care for systemd and docker logs

Simple, exclude them at the kube-system level:

```bash
kube-system.conf:
<match systemd.** docker>
  @type null
</match>

<match **>
  # all but systemd.** is still around
  # configure destination
</match>
```

### I want to use one destination but also want to use $labels to exclude a verbose pod/container from a single namespace

This is not possible. Instead, provide this config for the noisy namespace and configure other namespaces at the cost of some code duplication:
```bash
noisy-namespace.conf:
<match $labels(app=verbose-logger)>
  @type null
</match>


other-namespace.conf
<match **>
  @type ...
</match>
```

On the bright side, the configuration of `other-namespace` contains nothing specific to other-namespace and the same content can be used for all namespaces whose logs we need collected.

### I want to push logs from namespace demo to logz.io

```bash
demo.conf:
<match **>
  @type logzio_buffered
  endpoint_url https://listener.logz.io:8071?token=TOKEN&type=log-router
  output_include_time true
  output_include_tags true
  buffer_type    file
  buffer_path    /var/log/kube-system-logz.buf
  flush_interval 10s
  buffer_chunk_limit 1m
</match>
```

For details you should consult the plugin documentation.

### I want to push logs from namespace test to papertrail

```bash
test.conf:
<match **>
    @type papertrail
    papertrail_host YOUR_HOST.papertrailapp.com
    papertrail_port YOUR_PORT
    flush_interval 30
</match>
```

For details you should consult the plugin documentation.

### I want to validate my config file before using it as a configmap

The container comes with a file validation command. To use it put all your *.conf file in a directory. Use the namespace name for the filename.
Then use this one-liner, bind-mounting the folder and feeding it as a `DATASOURCE_DIR` env var:

```bash
docker run --entrypoint=/bin/validate-from-dir.sh \
    --net=host --rm \
    -v /path/to/config-folder:/workspace \
    -e DATASOURCE_DIR=/workspace \
    jvassev/kube-fluentd-operator:latest
```

It will run fluentd in dry-run mode and even catch incorrect plug-in usage.
This is so common that it' already captured as a script [validate-logging-config.sh](https://github.com/vmware/kube-fluentd-operator/blob/master/config-reloader/validate-logging-config.sh).
The preferred way to use it is to copy it to your project and invoke it like this:

```bash
validate-logging-config.sh path/to/folder

```

All `path/to/folder/*.conf` files will be validated. Check stderr and the exit code for errors.

### I want to use Fluentd @label to simplify processing

Use `<label>` as usual, the daemon ensures that label names are unique cluster-wide. For example to route several pods' logs to destination X, and ignore a few others you can use this:

```
<match $labels(app=foo)>
  @type label
  @label blackhole
</match>

<match $labels(app=bar)>
  @type label
  @label blackhole
</match>

<label @blackhole>
  <match **>
    @type null
  </match>
</label>

# at this point, foo and bar's logs are being handled by the @blackhole chain,
# the rest are still available for processing
<match **>
  @type ..
  # destination X
</match>
```

### I want to parse ingress-nginx access logs and send them to a different log aggregator

The ingress controller uses a format different than the plain Nginx. You can use this fragment to configure the namespace hosting the ingress-nginx controller:

```
<filter $labels(app=nginx-ingress, _container=nginx-ingress-controller)>
  @type parser

  format /(?<remote_addr>[^ ]*) - \[(?<proxy_protocol_addr>[^ ]*)\] - (?<remote_user>[^ ]*) \[(?<time>[^\]]*)\] "(?<method>\S+)(?: +(?<request>[^\"]*) +\S*)?" (?<code>[^ ]*) (?<size>[^ ]*) "(?<referer>[^\"]*)" "(?<agent>[^\"]*)" (?<request_length>[^ ]*) (?<request_time>[^ ]*) \[(?<proxy_upstream_name>[^ ]*)\] (?<upstream_addr>[^ ]*) (?<upstream_response_length>[^ ]*) (?<upstream_response_time>[^ ]*) (?<upstream_status>[^ ]*)/
  time_format %d/%b/%Y:%H:%M:%S %z
  key_name log
  reserve_data true
</filter>

<match **>
  # send the parsed access logs here
</match>
```

The above configuration assumes you're using the Helm charts for Nginx ingress. If not, make sure to the change the `app` and `_container` labels accordingly.

### I want to build a custom image with my secret fluentd plugin

Use the `vmware/kube-fluentd-operator:TAG` as a base and do any modification as usual.

### How can I be sure to use a valid path for the .pos and .buf files

.pos files store the progress of the upload process and .buf are used for local buffering. Colliding .pos/.buf paths can lead to races in Fluentd. As such, `kube-fluentd-operator` tries hard to rewrite such path-based parameters in predictable way. You only need to make sure they are unique for your namespace and `config-reloader` will take care to make them unique cluster-wide.

### I dont like the annotation name logging.csp.vmware.com/fluentd-configmap

Use `--annotation=acme.com/fancy-config` to use acme.com/fancy-config as annotation name. However, you'd also need to customize the Helm chart. Patches are welcome!

## Known issues

Currently space-delimited tags are not supported. For example, instead of `<filter a b>`, you need to use `<filter a>` and `<filter b>`.
This limitation will be addressed in a later version.

Some Fluentd plug-ins terminate the process if missconfigured. While we try hard to validate configuraion before applying it the S3 plugin will exit(1) on the first error it encounters in case the AWS credentials are invalid. As a workaround, don't use S3 if you cannot ensure it will be configured properly. To enforce it, build a custom image by un-installing the S3 plugin:

```Dockerfile
FROM ...

RUN fluent-gem uninstall fluent-plugin-s3
```

Then use the new image with helm:

```bash
helm install ... --set image.repository=acme.com/my-custom-image
```

## Releases

* 1.0.0: Initial version

## Resoures

* This plugin is used to provide kubernetes metadata https://github.com/fabric8io/fluent-plugin-kubernetes_metadata_filter
* This daemonset definition is used as a template: https://github.com/fluent/fluentd-kubernetes-daemonset/tree/master/docker-image/v0.12/debian-elasticsearch, however `kube-fluentd-operator` uses version 1.x version of fluentd and all the compatible plugin versions.
* This [Github issue](https://github.com/fabric8io/fluent-plugin-kubernetes_metadata_filter/issues/73) was used as an inspiration for the project. In particular it borrows the tag rewriting based on Kubernetes metadata to allow easier routing after that.

## Contributing

The kube-fluentd-operator project team welcomes contributions from the community. If you wish to contribute code and you have not
signed our contributor license agreement (CLA), our bot will update the issue when you open a Pull Request. For any
questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq). For more detailed information,
refer to [CONTRIBUTING.md](CONTRIBUTING.md).

## License
Redistribution and use in source and binary forms, with or without modification, are permitted provided that the following conditions are met:
*	Redistributions of source code must retain the above copyright notice, this list of conditions and the following disclaimer.
*	Redistributions in binary form must reproduce the above copyright notice, this list of conditions and the following disclaimer in the documentation and/or other materials provided with the distribution.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
