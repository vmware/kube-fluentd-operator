# kube-fluentd-operator (KFO)

[![Go Report Card](https://goreportcard.com/badge/github.com/vmware/kube-fluentd-operator)](https://goreportcard.com/report/github.com/vmware/kube-fluentd-operator)

## Overview

Kubernetes Fluentd Operator (KFO) is a Fluentd config manager with batteries included, config validation, no needs to restart, with sensible defaults and best practices built-in. Use Kubernetes labels to filter/route logs per namespace!

_kube-fluentd-operator_ configures Fluentd in a Kubernetes environment. It compiles a Fluentd configuration from configmaps (one per namespace) - similar to how an Ingress controller would compile nginx configuration from several Ingress resources. This way only one instance of Fluentd can handle all log shipping for an entire cluster and the cluster admin does NOT need to coordinate with namespace admins.

Cluster administrators set up Fluentd only once and namespace owners can configure log routing as they wish. _KFO_ will re-configure Fluentd accordingly and make sure logs originating from a namespace will not be accessible by other tenants/namespaces.

_KFO_ also extends the Fluentd configuration language making it possible to refer to pods based on their labels and the container name pattern. This enables for very fined-grained targeting of log streams for the purpose of pre-processing before shipping. Writing a custom processor, adding a new Fluentd plugin, or writing a custom Fluentd plugin allow KFO to be extendable for any use case and any external logging ingestion system.

Finally, it is possible to ingest logs from a file on the container filesystem. While this is not recommended, there are still legacy or misconfigured apps that insist on logging to the local filesystem.

## Try it out

The easiest way to get started is using the Helm chart. Official images are not published yet, so you need to pass the image.repository and image.tag manually:

```bash
git clone git@github.com:vmware/kube-fluentd-operator.git
helm install kfo ./kube-fluentd-operator/charts/log-router \
  --set rbac.create=true \
  --set image.tag=v1.18.0 \
  --set image.repository=vmware/kube-fluentd-operator
```

Alternatively, deploy the Helm chart from a Github release:

```bash
CHART_URL='https://github.com/vmware/kube-fluentd-operator/releases/download/v1.18.0/log-router-0.4.0.tgz'

helm install kfo ${CHART_URL} \
  --set rbac.create=true \
  --set image.tag=v1.18.0 \
  --set image.repository=vmware/kube-fluentd-operator
```

Then create a namespace `demo` and a configmap describing where all logs from `demo` should go to. The configmap must contain an entry called "fluent.conf". Finally, point the kube-fluentd-operator to this configmap using annotations.

```bash
kubectl create ns demo

cat > fluent.conf << EOF
<match **>
  @type null
</match>
EOF

# Create the configmap with a single entry "fluent.conf"
kubectl create configmap fluentd-config --namespace demo --from-file=fluent.conf=fluent.conf


# The following step is optional: the fluentd-config is the default configmap name.
# kubectl annotate namespace demo logging.csp.vmware.com/fluentd-configmap=fluentd-config

```

In a minute, this configuration would be translated to something like this:

```xml
<match demo.**>
  @type null
</match>
```

Even though the tag `**` was used in the `<match>` directive, the kube-fluentd-operator correctly expands this to `demo.**`. Indeed, if another tag which does not start with `demo.` was used, it would have failed validation. Namespace admins can safely assume that they has a dedicated Fluentd for themselves.

All configuration errors are stored in the annotation `logging.csp.vmware.com/fluentd-status`. Try replacing `**` with an invalid tag like 'hello-world'. After a minute, verify that the error message looks like this:

```bash
# extract just the value of logging.csp.vmware.com/fluentd-status
kubectl get ns demo -o jsonpath='{.metadata.annotations.logging\.csp\.vmware\.com/fluentd-status}'
bad tag for <match>: hello-world. Tag must start with **, $thisns or demo
```

When the configuration is made valid again the `fluentd-status` is set to "".

To see kube-fluentd-operator in action you need a cloud log collector like logz.io, papertrail or ELK accessible from the K8S cluster. A simple logz.io configuration looks like this (replace TOKEN with your customer token):

```xml
<match **>
   @type logzio_buffered
   endpoint_url https://listener.logz.io:8071?token=$TOKEN
</match>
```

## Build

Get the code using `go get` or git clone this repo:

```bash
go get -u github.com/vmware/kube-fluentd-operator/config-reloader
cd $GOPATH/src/github.com/vmware/kube-fluentd-operator

# build a base-image
cd base-image && make build-image

# build helm chart
cd charts/log-router && make helm-package

# build the daemon
cd config-reloader
make install
make build-image

# run with mock data (loaded from the examples/ folder)
make run-once-fs

# run with mock data in a loop (may need to ctrl+z to exit)
make run-loop-fs

# inspect what is generated from the above command
ls -l tmp/
```

### Project structure

- `charts/log-router`: Builds the Helm chart
- `base-image`: Builds a Fluentd 1.2.x image with a curated list of plugins
- `config-reloader`: Builds the daemon that generates fluentd configuration files

### Config-reloader

This is where interesting work happens. The [dependency graph](config-reloader/godepgraph.png) shows the high-level package interaction and general dataflow.

- `config`: handles startup configuration, reading and validation
- `datasource`: fetches Pods, Namespaces, ConfigMaps from Kubernetes
- `fluentd`: parses Fluentd config files into an object graph
- `processors`: walks this object graph doing validations and modifications. All features are implemented as chained `Processor` subtypes
- `generator`: serializes the processed object graph to the filesystem for Fluentd to read
- `controller`: orchestrates the high-level `datasource` -> `processor` -> `generator` pipeline.

### How does it work

It works be rewriting the user-provided configuration. This is possible because _kube-fluentd-operator_ knows about the kubernetes cluster, the current namespace and
also has some sensible defaults built in. To get a quick idea what happens behind the scenes consider this configuration deployed in a namespace called `monitoring`:

```xml
<filter $labels(server=apache)>
  @type parser
  <parse>
    @type apache2
  </parse>
</filter>

<filter $labels(app=django)>
  @type detect_exceptions
  language python
</filter>

<match **>
  @type es
</match>
```

It gets processed into the following configuration which is then fed to Fluentd:

```xml
<filter kube.monitoring.*.*>
  @type record_transformer
  enable_ruby true

  <record>
    kubernetes_pod_label_values ${record["kubernetes"]["labels"]["app"]&.gsub(/[.-]/, '_') || '_'}.${record["kubernetes"]["labels"]["server"]&.gsub(/[.-]/, '_') || '_'}
  </record>
</filter>

<match kube.monitoring.*.*>
  @type rewrite_tag_filter

  <rule>
    key kubernetes_pod_label_values
    pattern ^(.+)$
    tag ${tag}._labels.$1
  </rule>
</match>

<filter kube.monitoring.*.*.**>
  @type record_modifier
  remove_keys kubernetes_pod_label_values
</filter>

<filter kube.monitoring.*.*._labels.*.apache _proc.kube.monitoring.*.*._labels.*.apache>
  @type parser
  <parse>
    @type apache2
  </parse>
</filter>

<match kube.monitoring.*.*._labels.django.*>
  @type rewrite_tag_filter

  <rule>
    invert true
    key _dummy
    pattern /ZZ/
    tag 3bfd045d94ce15036a8e3ff77fcb470e0e02ebee._proc.${tag}
  </rule>
</match>

<match 3bfd045d94ce15036a8e3ff77fcb470e0e02ebee._proc.kube.monitoring.*.*._labels.django.*>
  @type detect_exceptions
  remove_tag_prefix 3bfd045d94ce15036a8e3ff77fcb470e0e02ebee
  stream container_info
</match>

<match kube.monitoring.*.*._labels.*.* _proc.kube.monitoring.*.*._labels.*.*>
  @type es
</match>
```

## Configuration

### Basic usage

To give the illusion that every namespace runs a dedicated Fluentd the user-provided configuration is post-processed. In general, expressions starting with `$` are macros that are expanded. These two directives are equivalent: `<match **>`, `<match $thisns>`. Almost always, using the `**` is the preferred way to match logs: this way you can reuse the same configuration for multiple namespaces.

### The admin namespace

Kube-fluentd-operator defines one namespace to be the _admin_ namespace. By default this is set to `kube-system`. The _admin_ namespace is treated differently. Its configuration is not processed further as it is assumed only the cluster admin can manipulate resources in this namespace. If you don't plan to use any of the advanced features described bellow, you can just route all logs from all namespaces using this snippet in the _admin_ namespace:

```xml
<match **>
 @type ...
 # destination configuration omitted
</match>
```

`**` in this context is not processed and it means _literally_ everything.

Fluentd assumes it is running in a distro with systemd and generates logs with these Fluentd tags:

- `systemd.{unit}`: the journal of a systemd unit, for example `systemd.docker.service`
- `docker`: all docker logs, not containers. If systemd is used, the docker logs are in `systemd.docker.service`
- `k8s.{component}`: logs from a K8S component, for example `k8s.kube-apiserver`
- `kube.{namespace}.{pod_name}.{container_name}`: a log originating from (namespace, pod, container)

As the _admin_ namespace is processed first, a match-all directive would consume all logs and any other namespace configuration will become irrelevant (unless `<copy>` is used).
A recommended configuration for the _admin_ namespace is this one (assuming it is set to `kube-system`) - it captures all but the user namespaces' logs:

```xml
<match systemd.** kube.kube-system.** k8s.** docker>
  # all k8s-internal and OS-level logs

  # destination config omitted...
</match>
```

Note the `<match systemd.**` syntax. A single `*` would not work as the tag is the full name - including the unit type, for example _systemd.nginx.service_

### Using the $labels macro

A very useful feature is the `<filter>` and the `$labels` macro to define parsing at the namespace level. For example, the config-reloader container uses the `logfmt` format. This makes it easy to use structured logging and ingest json data into a remote log ingestion service.

```xml
<filter $labels(app=log-router, _container=reloader)>
  @type parser
  reserve_data true
  <parse>
    @type logfmt
  </parse>
</filter>

<match **>
  @type logzio_buffered
  # destination config omitted
</match>
```

The above config will pipe all logs from the pods labelled with `app=log-router` through a [logfmt](https://github.com/vmware/kube-fluentd-operator/blob/master/base-image/plugins/parser_logfmt.rb) parser before sending them to logz.io. Again, this configuration is valid in any namespace. If the namespace doesn't contain any `log-router` components then the `<filter>` directive is never activated. The `_container` is sort of a "meta" label and it allows for targeting the log stream of a specific container in a multi-container pod.

If you use [Kubernetes recommended labels](https://kubernetes.io/docs/concepts/overview/working-with-objects/common-labels/) for the pods and deployments, then KFO will rewrite `.` characters into `_`.

For example, let's assume the following labels exist in the fluentd-config in the `testing` namespace:

This label `$labels(_container=nginx-ingress-controller)` will filter by container name pattern. The label will convert to this for example: `kube.testing.*.nginx-ingress-controller._labels.*.*.`

This label `$labels(app.kubernetes.io/name=nginx-ingress, _container=nginx-ingress-controller)` converts to this `kube.testing.*.nginx-ingress-controller._labels.*.nginx_ingress`.

This label `$labels(app.kubernetes.io/name=nginx-ingress)` converts to this `$labels(kube.testing.*.*._labels.*.nginx_ingress)`.

This fluentd configmap in the `testing` namespace:

```xml
<filter **>
  @type concat
  timeout_label @DISTILLERY_TYPES
  key message
  stream_identity_key cont_id
  multiline_start_regexp /^(\d{4}-\d{1,2}-\d{1,2} \d{1,2}:\d{1,2}:\d{1,2}|\[\w+\]\s|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b|=\w+ REPORT====|\d{2}\:\d{2}\:\d{2}\.\d{3})/
  flush_interval 10
</filter>

<match **>
  @type relabel
  @label @DISTILLERY_TYPES
</match>

<label @DISTILLERY_TYPES>
  <filter $labels(app_kubernetes_io/name=kafka)>
    @type parser
    key_name log
    format json
    reserve_data true
    suppress_parse_error_log true
  </filter>

  <filter $labels(app.kubernetes.io/name=nginx-ingress, _container=controller)>
    @type parser
    key_name log

    <parse>
      @type json
      reserve_data true
      time_format %FT%T%:z
      emit_invalid_record_to_error false
    </parse>
  </filter>

  <match $labels(tag=noisy)>
    @type null
  </match>
</label>
```

will be rewritten inside of KFO pods as this:

```xml
<filter kube.testing.**>
  @type concat
  flush_interval 10
  key message
  multiline_start_regexp /^(\d{4}-\d{1,2}-\d{1,2} \d{1,2}:\d{1,2}:\d{1,2}|\[\w+\]\s|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}\b|=\w+ REPORT====|\d{2}\:\d{2}\:\d{2}\.\d{3})/
  stream_identity_key cont_id
  timeout_label @-DISTILLERY_TYPES-0e93f964a5b5f1760278744f1adf55d58d0e78ba
</filter>

<match kube.testing.**>
  @label @-DISTILLERY_TYPES-0e93f964a5b5f1760278744f1adf55d58d0e78ba
  @type relabel
</match>

<match kube.testing.**>
  @label @-DISTILLERY_TYPES-0e93f964a5b5f1760278744f1adf55d58d0e78ba
  @type null
</match>

<label @-DISTILLERY_TYPES-0e93f964a5b5f1760278744f1adf55d58d0e78ba>
  <filter kube.testing.*.*._labels.*.kafka.*>
    @type parser
    format json
    key_name log
    reserve_data true
    suppress_parse_error_log true
  </filter>
  <filter kube.testing.*.controller._labels.nginx_ingress.*.*>
    @type parser
    key_name log

    <parse>
      @type json
      emit_invalid_record_to_error false
      reserve_data true
      time_format %FT%T%:z
    </parse>
  </filter>
  <match kube.testing.*.*._labels.*.*.noisy>
    @type null
  </match>
</label>
```

All plugins that change the fluentd tag are disabled for security reasons. Otherwise a rogue configuration may divert other namespace's logs to itself by prepending its name to the tag.

### Ingest logs from a file in the container

The only allowed `<source>` directive is of type `mounted-file`. It is used to ingest a log file from a container on an `emptyDir`-mounted volume:

```xml
<source>
  @type mounted-file
  path /var/log/welcome.log
  labels app=grafana, _container=test-container
  <parse>
    @type none
  </parse>
</source>
```

The `labels` parameter is similar to the `$labels` macro and helps the daemon locate all pods that might log to the given file path. The `<parse>` directive is optional and if omitted the default `@type none` will be used. If you know the format of the log file you can explicitly specify it, for example `@type apache2` or `@type json`.

The above configuration would translate at runtime to something similar to this:

```xml
<source>
  @type tail
  path /var/lib/kubelet/pods/723dd34a-4ac0-11e8-8a81-0a930dd884b0/volumes/kubernetes.io~empty-dir/logs/welcome.log
  pos_file /var/log/kfotail-7020a0b821b0d230d89283ba47d9088d9b58f97d.pos
  read_from_head true
  tag kube.kfo-test.welcome-logger.test-container

  <parse>
    @type none
  </parse>
</source>
```

### Dealing with multi-line exception stacktraces (since v1.3.0)

Most log streams are line-oriented. However, stacktraces always span multiple lines. _kube-fluentd-operator_ integrates stacktrace processing using the [fluent-plugin-detect-exceptions](https://github.com/GoogleCloudPlatform/fluent-plugin-detect-exceptions). If a Java-based pod produces stacktraces in the logs, then the stacktraces can be collapsed in a single log event like this:

```xml
<filter $labels(app=jpetstore)>
  @type detect_exceptions
  # you can skip language in which case all possible languages will be tried: go, java, python, ruby, etc...
  language java
</filter>

# The rest of the configuration stays the same even though quite a lot of tag rewriting takes place

<match **>
 @type es
</match>
```

Notice how `filter` is used instead of `match` as described in [fluent-plugin-detect-exceptions](https://github.com/GoogleCloudPlatform/fluent-plugin-detect-exceptions). Internally, this filter is translated into several `match` directives so that the end user doesn't need to bother with rewriting the Fluentd tag.

Also, users don't need to bother with setting the correct `stream` parameter. _kube-fluentd-operator_ generates one internally based on the container id and the stream.

### Reusing output plugin definitions (since v1.6.0)

Sometimes you only have a few valid options for log sinks: a dedicated S3 bucket, the ELK stack you manage, etc. The only flexibility you're after is letting namespace owners filter and parse their logs. In such cases you can abstract over an output plugin configuration - basically reducing it to a simple name which can be referenced from any namespace. For example, let's assume you have an S3 bucket for a "test" environment and you use logz.io for a "staging" environment. The first thing you do is define these two output in the _admin_ namespace:

```xml
admin-ns.conf:
<match systemd.** docker kube.kube-system.** k8s.**>
  @type logzio_buffered
  endpoint_url https://listener.logz.io:8071?token=$TOKEN
</match>

<plugin test>
  @type s3
  aws_key_id  YOUR_AWS_KEY_ID
  aws_sec_key YOUR_AWS_SECRET_KEY
  s3_bucket   YOUR_S3_BUCKET_NAME
  s3_region   AWS_REGION
</plugin>

<plugin staging>
  @type logzio_buffered
  endpoint_url https://listener.logz.io:8071?token=$TOKEN
</plugin>
```

In the above example for the admin configuration, the `match` directive is first defined to direct where to send logs for the `systemd`, `docker`, `kube-system`, and kubernetes control plane components. Below the `match` directive we have defined the `plugin` directives which define the log sinks that can be reused by namespace configurations.

A namespace can refer to the `staging` and `test` plugins oblivious to the fact where exactly the logs end up:

```xml
acme-test.conf
<match **>
  @type test
</match>


acme-staging.conf
<match **>
  @type staging
</match>
```

kube-fluentd-operator will insert the content of the `plugin` directive in the `match` directive. From then on, regular validation and postprocessing takes place.

### Retagging based on log contents (since v1.12.0)

Sometimes you might need to split a single log stream to perform different processing based on the contents of one of the fields. To achieve this you can use the `retag` plugin that allows to specify a set of rules that match regular expressions against the specified fields. If one of the rules matches, the log is re-emitted with a new namespace-unique tag based on the specified tag.

Logs that are emitted by this plugin can be consequently filtered and processed by using the `$tag` macro when specifiying the tag:

```xml
<match $labels(app=apache)>
  @type retag
  <rule>
    key message
    pattern /^(ERROR) .*$/
    tag notifications.$1 # refer to a capturing group using $number
  </rule>
  <rule>
    key message
    pattern /^(FATAL) .*$/
    tag notifications.$1
  </rule>
  <rule>
    key message
    pattern /^(ERROR)|(FATAL) .*$/
    tag notifications.other
    invert true # rewrite tag when unmatch pattern
  </rule>
</match>

<filter $tag(notifications.ERROR)>
  # perform some extra processing
</filter>

<filter $tag(notifications.FATAL)>
  # perform different processing
</filter>

<match $tag(notifications.**)>
  # send to common output plugin
</match>
```

_kube-fluentd-operator_ ensures that tags specified using the `$tag` macro never conflict with tags from other namespaces, even if the tag itself is equivalent.

### Sharing logs between namespaces

By default, you can consume logs only from your namespaces. Often it is useful for multiple namespaces (tenants) to get access to the logs streams of a shared resource (pod, namespace). _kube-fluentd-operator_ makes it possible using two constructs: the source namespace expresses its intent to share logs with a destination namespace and the destination namespace expresses its desire to consume logs from a source. As a result logs are streamed only when both sides agree.

A source namespace can share with another namespace using the `@type share` macro:

producer namespace configuration:

```xml
<match $labels(msg=nginx-ingress)>
  @type copy
  <store>
    @type share
    # share all logs matching the labels with the namespace "consumer"
    with_namespace consumer
  </store>
</match>
```

consumer namespace configuration:

```xml
# use $from(producer) to get all shared logs from a namespace called "producer"
<label @$from(producer)>
  <match **>
    # process all shared logs here as usual
  </match>
</match>
```

The consuming namespace can use the usual syntax inside the `<label @$from...>` directive. The fluentd tag is being rewritten as if the logs originated from the same namespace.

The producing namespace need to wrap `@type share` within a `<store>` directive. This is done on purpose as it is very easy to just redirect the logs to the destination namespace and lose them. The `@type copy` clones the whole stream.

### Log metadata

Often you run mulitple Kubernetes clusters but you need to aggregate all logs to a single destination. To distinguish between different sources, `kube-fluentd-operator` can attach arbitrary metadata to every log event.
The metadata is nested under a key chosen with `--meta-key`. Using the helm chart, metadata can be enabled like this:

```bash
helm install ... \
  --set meta.key=metadata \
  --set meta.values.region=us-east-1 \
  --set meta.values.env=staging \
  --set meta.values.cluster=legacy
```

Every log event, be it from a pod, mounted-file or a systemd unit, will now carry this metadata:

```json
{
  "metadata": {
    "region": "us-east-1",
    "env": "staging",
    "cluster": "legacy"
  }
}
```

All logs originating from a file look exactly as all other Kubernetes logs. However, their `stream` field is not set to `stdout` but to the path to the source file:

```json
{
  "message": "Some message from the welcome-logger pod",
  "stream": "/var/log/welcome.log",
  "kubernetes": {
    "container_name": "test-container",
    "host": "ip-11-11-11-11.us-east-2.compute.internal",
    "namespace_name": "kfo-test",
    "pod_id": "723dd34a-4ac0-11e8-8a81-0a930dd884b0",
    "pod_name": "welcome-logger",
    "labels": {
      "msg": "welcome",
      "test-case": "b"
    },
    "namespace_labels": {}
  },
  "metadata": {
    "region": "us-east-2",
    "cluster": "legacy",
    "env": "staging"
  }
}
```

### Go templting

The `ConfigMap` holding the fluentd configuration can be templated using `go` templting, you can use this for example to get a value from another kubernetes resource, like a secret, for example:

```yaml
kind: ConfigMap
apiVersion: v1
metadata:
  annotations: {}
  name: fluentd-config
  namespace: my-namespace
data:
  fluent.conf: |
    {{- $s := k8sLookup "Secret.v1" "my-namespace" "my-secret" -}}
    <match **>
      @type logzio_buffered
      endpoint_url https://listener.logz.io:8071?token={{ $s.data.token }}&type=log-router
      output_include_time true
      output_include_tags false
      http_idle_timeout 10

      <buffer>
        @type file
        path /var/log/my_namespace.log.buf
        flush_thread_count 4
        flush_interval 10s
        chunk_limit_size 16m
        queue_limit_length 4096
      </buffer>
    </match>
```

### Custom resource definition(CRD) support (since v1.13.0)

Custom resources are introduced from v1.13.0 release onwards. It allows to have a dedicated resource for fluentd configurations, which enables to manage them in a more consistent way and move away from the generic ConfigMaps.
It is possible to create configs for a new application simply by attaching a FluentdConfig resource to the application manifests, rather than using a more generic ConfigMap with specific names and/or labels.

```xml
apiVersion: logs.vdp.vmware.com/v1beta1
kind: FluentdConfig
metadata:
  name: fd-config
spec:
  fluentconf: |
    <match kube.ns.**>
      @type relabel
      @label @NOTIFICATIONS
    </match>

    <label @NOTIFICATIONS>
     <match **>
       @type null
     </match>
    </label>
```

The "crd" has been introduced as a new datasource, configurable through the helm chart values, to allow users that are currently set up with ConfigMaps and do not want to perform the switchover to FluentdConfigs, to be able to keep on using them. The config-reloader has been equipped with the capability of installing the CRD at startup if requested, so no manual actions to enable it on the cluster are needed.
The existing configurations though ConfigMaps can be migrated to CRDs through the following migration flow

- A new user, who is installing kube-fluentd-operator for the first time, should set the datasource: crd option in the chart. This enables the crd support
- A user who is already using kube-fluentd-operator with either datasource: default or datasource: multimap will have update to the new chart and set the 'crdMigrationMode' property to 'true'. This enables the config-reloader to launch with the crd datasource and the legacy datasource (either default or multimap depending on what was configured in the datasource property). The user can slowly migrate one by one all configmap resources to the corresponding fluentdconfig resources. When the migration is complete, the Helm release can be upgraded by changing the 'crdMigrationMode' property to 'false' and switching the datasource property to 'crd'. This will effectively disable the legacy datasource and set the config-reloader to only watch fluentdconfig resources.

## Tracking Fluentd version

This projects tries to keep up with major releases for [Fluentd docker image](https://github.com/fluent/fluentd-docker-image/).

| Fluentd version | Operator version |
| --------------- | ---------------- |
| 0.12.x          | 1.0.0            |
| 1.15.3          | 1.17.1           |
| 1.16.1          | 1.17.6           |
| 1.16.1          | 1.18.0           |

## Plugins in latest release (1.18.0)

`kube-fluentd-operator` aims to be easy to use and flexible. It also favors sending logs to multiple destinations using `<copy>` and as such comes with many plugins pre-installed:

- fluentd (1.16.1)
- fluent-plugin-amqp (0.14.0)
- fluent-plugin-azure-loganalytics (0.7.0)
- fluent-plugin-cloudwatch-logs (0.14.3)
- fluent-plugin-concat (2.5.0)
- fluent-plugin-datadog (0.14.2)
- fluent-plugin-elasticsearch (5.3.0)
- fluent-plugin-opensearch (1.1.0)
- fluent-plugin-gelf-hs (1.0.8)
- fluent-plugin-google-cloud (0.13.0) - forked to allow fluentd v1.14.x
- fluent-plugin-grafana-loki (1.2.20)
- fluent-plugin-grok-parser (2.6.2)
- fluent-plugin-json-in-json-2 (1.0.2)
- fluent-plugin-kafka (0.18.1)
- fluent-plugin-kinesis (3.4.2)
- fluent-plugin-kubernetes_metadata_filter (3.2.0)
- fluent-plugin-kubernetes_sumologic (2.4.2)
- fluent-plugin-kubernetes (0.3.1)
- fluent-plugin-logentries (0.2.10)
- fluent-plugin-logzio (0.0.22)
- fluent-plugin-mail (0.3.0)
- fluent-plugin-mongo (1.5.0)
- fluent-plugin-multi-format-parser (1.0.0)
- fluent-plugin-papertrail (0.2.8)
- fluent-plugin-prometheus (2.1.0)
- fluent-plugin-record-modifier (2.1.0)
- fluent-plugin-record-reformer (0.9.1)
- fluent-plugin-redis (0.3.5)
- fluent-plugin-remote_syslog (1.0.0)
- fluent-plugin-rewrite-tag-filter (2.4.0)
- fluent-plugin-route (1.0.0)
- fluent-plugin-s3 (1.7.2)
- fluent-plugin-splunk-hec (1.3.1)
- fluent-plugin-splunkhec (2.3)
- fluent-plugin-sumologic_output (1.7.3)
- fluent-plugin-systemd (1.0.5)
- fluent-plugin-uri-parser (0.3.0)
- fluent-plugin-verticajson (0.0.6)
- fluent-plugin-vmware-loginsight (1.4.1)
- fluent-plugin-vmware-log-intelligence (2.0.8)
- fluent-plugin-mysqlslowquery (0.0.9)
- fluent-plugin-throttle (0.0.5)
- fluent-plugin-webhdfs (1.5.0)
- fluent-plugin-detect-exceptions (0.0.15)

When customizing the image be careful not to uninstall plugins that are used internally to implement the macros.

If you need other destination plugins you are welcome to contribute a patch or just create an issue.

## Synopsis

The config-reloader binary is the one that listens to changes in K8S and generates Fluentd files. It runs as a daemonset and is not intended to interact with directly. The synopsis is useful when trying to understand the Helm chart or just hacking.

```txt
usage: config-reloader [<flags>]

Regenerates Fluentd configs based Kubernetes namespace annotations against templates, reloading
Fluentd if necessary

Flags:
  --help                        Show context-sensitive help (also try --help-long and
                                --help-man).
  --version                     Show application version.
  --master=""                   The Kubernetes API server to connect to (default: auto-detect)
  --kubeconfig=""               Retrieve target cluster configuration from a Kubernetes
                                configuration file (default: auto-detect)
  --datasource=default          Datasource to use (default|fake|fs|multimap|crd)
  --crd-migration-mode          Enable the crd datasource together with the current datasource to facilitate the migration (used only with --datasource=default|multimap)
  --fs-dir=FS-DIR               If datasource=fs is used, configure the dir hosting the files
  --interval=60                 Run every x seconds
  --allow-file                  Allow @type file for namespace configuration
  --id="default"                The id of this deployment. It is used internally so that two
                                deployments don't overwrite each other's data
  --fluentd-rpc-port=24444      RPC port of Fluentd
  --log-level="info"            Control verbosity of config-reloader logs
  --fluentd-loglevel="info"     Control verbosity of fluentd logs
  --buffer-mount-folder=""      Folder in /var/log/{} where to create all fluentd buffers
  --annotation="logging.csp.vmware.com/fluentd-configmap"
                                Which annotation on the namespace stores the configmap name?
  --default-configmap="fluentd-config"
                                Read the configmap by this name if namespace is not annotated.
                                Use empty string to suppress the default.
  --status-annotation="logging.csp.vmware.com/fluentd-status"
                                Store configuration errors in this annotation, leave empty to
                                turn off
  --kubelet-root="/var/lib/kubelet/"
                                Kubelet root dir, configured using --root-dir on the kubelet
                                service
  --namespaces=NAMESPACES ...   List of namespaces to process. If empty, processes all namespaces
  --templates-dir="/templates"  Where to find templates
  --output-dir="/fluentd/etc"   Where to output config files
  --meta-key=META-KEY           Attach metadat under this key
  --meta-values=META-VALUES     Metadata in the k=v,k2=v2 format
  --fluentd-binary=FLUENTD-BINARY
                                Path to fluentd binary used to validate configuration
  --prometheus-enabled          Prometheus metrics enabled (default: false)
  --admin-namespace="kube-system"
                                The namespace to be treated as admin namespace

```

## Helm chart

| Parameter                    | Description                                                                                                          | Default                        |
| ---------------------------- | -------------------------------------------------------------------------------------------------------------------- | ------------------------------ |
| `rbac.create`                | Create a serviceaccount+role, use if K8s is using RBAC                                                               | `false`                        |
| `serviceAccountName`         | Reuse an existing service account                                                                                    | `""`                           |
| `defaultConfigmap`           | Read the configmap by this name if the namespace is not annotated                                                    | `"fluentd-config"`             |
| `image.repositiry`           | Repository                                                                                                           | `vmware/kube-fluentd-operator` |
| `image.tag`                  | Image tag                                                                                                            | `latest`                       |
| `image.pullPolicy`           | Pull policy                                                                                                          | `Always`                       |
| `image.pullSecret`           | Optional pull secret name                                                                                            | `""`                           |
| `logLevel`                   | Default log level for config-reloader                                                                                | `info`                         |
| `fluentdLogLevel`            | Default log level for fluentd                                                                                        | `info`                         |
| `bufferMountFolder`          | Folder in /var/log/{} where to create all fluentd buffers                                                            | `""`                           |
| `kubeletRoot`                | The home dir of the kubelet, usually set using `--root-dir` on the kubelet                                           | `/var/lib/kubelet`             |
| `namespaces`                 | List of namespaces to operate on. Empty means all namespaces                                                         | `[]`                           |
| `interval`                   | How often to check for config changes (seconds)                                                                      | `45`                           |
| `meta.key`                   | The metadata key (optional)                                                                                          | `""`                           |
| `meta.values`                | Metadata to use for the key                                                                                          | `{}`                           |
| `extraVolumes`               | Extra volumes                                                                                                        |                                |
| `fluentd.extraVolumeMounts`  | Mount extra volumes for the fluentd container, required to mount ssl certificates when elasticsearch has tls enabled |                                |
| `fluentd.resources`          | Resource definitions for the fluentd container                                                                       | `{}`                           |
| `fluentd.extraEnv`           | Extra env vars to pass to the fluentd container                                                                      | `{}`                           |
| `reloader.extraVolumeMounts` | Mount extra volumes for the reloader container                                                                       |                                |
| `reloader.resources`         | Resource definitions for the reloader container                                                                      | `{}`                           |
| `reloader.extraEnv`          | Extra env vars to pass to the reloader container                                                                     | `{}`                           |
| `tolerations`                | Pod tolerations                                                                                                      | `[]`                           |
| `updateStrategy`             | UpdateStrategy for the daemonset. Leave empty to get the K8S' default (probably the safest choice)                   | `{}`                           |
| `podAnnotations`             | Pod annotations for the daemonset                                                                                    |                                |
| `adminNamespace`             | The namespace to be treated as admin namespace                                                                       | `kube-system`                  |

## Cookbook

### I want to use one destination for everything

Simple, define configuration only for the _admin_ namespace (by default `kube-system`):

```bash
kube-system.conf:
<match **>
  # configure destination here
</match>
```

### I dont't care for systemd and docker logs

Simple, exclude them at the _admin_ namespace level (by default `kube-system`):

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

### I want to use one destination but also want to just exclude a few pods

It is not possible to handle this globally. Instead, provide this config for the noisy namespace and configure other namespaces at the cost of some code duplication:

```xml
noisy-namespace.conf:
<match $labels(app=verbose-logger)>
  @type null
</match>

# all other logs are captured here
<match **>
  @type ...
</match>
```

On the bright side, the configuration of `noisy-namespace` contains nothing specific to noisy-namespace and the same content can be used for all namespaces whose logs we need collected.

### I am getting errors "namespaces is forbidden: ... cannot list namespaces at the cluster scope"

Your cluster is running under RBAC. You need to enable a serviceaccount for the log-router pods. It's easy when using the Helm chart:

```bash
helm install ./charts/log-router --set rbac.create=true ...
```

### I have a legacy container that logs to /var/log/httpd/access.log

First you need version 1.1.0 or later. At the namespace level you need to add a `source` directive of type `mounted-file`:

```xml
<source>
  @type mounted-file
  path /var/log/httpd/access.log
  labels app=apache2
  <parse>
    @type apache2
  </parse>
</source>

<match **>
  # destination config omitted
</match>
```

The type `mounted-file` is again a macro that is expanded to a `tail` plugin. The `<parse>` directive is optional and if not set a `@type none` will be used instead.

In order for this to work the pod must define a mount of type `emptyDir` at `/var/log/httpd` or any of it parent folders. For example, this pod definition is part of the test suite (it logs to /var/log/hello.log):

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: hello-logger
  namespace: kfo-test
  labels:
    msg: hello
spec:
  containers:
    - image: ubuntu
      name: greeter
      command:
        - bash
        - -c
        - while true; do echo `date -R` [INFO] "Random hello number $((var++)) to file"; sleep 2; [[ $(($var % 100)) == 0 ]] && :> /var/log/hello.log ;done > /var/log/hello.log
      volumeMounts:
        - mountPath: /var/log
          name: logs
  volumes:
    - name: logs
      emptyDir: {}
```

To get the hello.log ingested by Fluentd you need at least this in the configuration for `kfo-test` namespace:

```xml
<source>
  @type mounted-file
  # need to specify the path on the container filesystem
  path /var/log/hello.log

  # only look at pods labeled this way
  labels msg=hello
  <parse>
    @type none
  </parse>
</source>

<match $labels(msg=hello)>
  # store the hello.log somewhere
  @type ...
</match>
```

### I want to push logs from namespace `demo` to logz.io

```xml
demo.conf:
<match **>
  @type logzio_buffered
  endpoint_url https://listener.logz.io:8071?token=TOKEN&type=log-router
  output_include_time true
  output_include_tags true
  <buffer>
    @type memory
    flush_thread_count 4
    flush_interval 3s
    queue_limit_length 4096
  </buffer>
</match>
```

For details you should consult the plugin documentation.

### I want to push logs to a remote syslog server

The built-in `remote_syslog` plugin cannot be used as the fluentd tag may be longer than 32 bytes. For this reason there is a `truncating_remote_syslog` plugin that shortens the tag to the allowed limit. If you are currently using the `remote_syslog` output plugin you only need to change a single line:

```xml
<match **>
  # instead of "remote_syslog"
  @type truncating_remote_syslog

  # the usual config for remote_syslog
</match>
```

To get the general idea how truncation works, consider this table:

| Original Tag                                  | Truncated tag                      |
| --------------------------------------------- | ---------------------------------- |
| `kube.demo.test.test`                         | `demo.test.test`                   |
| `kube.demo.nginx-65899c769f-5zj6d.nginx`      | `demo.nginx-65899c769f-5zj*.nginx` |
| `kube.demo.test.nginx11111111._lablels.hello` | `demo.test.nginx11111111`          |

### I want to push logs to Humio

Humio speaks the elasticsearh protocol so configuration is pretty similar to Elasticsearch. The example bellow is based on https://github.com/humio/kubernetes2humio/blob/master/fluentd/docker-image/fluent.conf.

```xml
<match **>
  @type elasticsearch
  include_tag_key false

  host "YOUR_HOST"
  path "/api/v1/dataspaces/YOUR_NAMESPACE/ingest/elasticsearch/"
  scheme "https"
  port "443"

  user "YOUR_KEY"
  password ""

  logstash_format true

  reload_connections "true"
  logstash_prefix "fluentd:kubernetes2humio"
  buffer_chunk_limit 1M
  buffer_queue_limit 32
  flush_interval 1s
  max_retry_wait 30
  disable_retry_limit
  num_threads 8
</match>
```

### I want to push logs to papertrail

```xml
test.conf:
<match **>
    @type papertrail
    papertrail_host YOUR_HOST.papertrailapp.com
    papertrail_port YOUR_PORT
    flush_interval 30
</match>
```

### I want to push logs to an ELK cluster

```xml
<match ***>
  @type elasticsearch
  host ...
  port ...
  index_name ...
  # many options available
</match>
```

For details you should consult the plugin documentation.

### I want to validate my config file before using it as a configmap

The container comes with a file validation command. To use it put all your \*.conf file in a directory. Use the namespace name for the filename. Then use this one-liner, bind-mounting the folder and feeding it as a `DATASOURCE_DIR` env var:

```bash
docker run --entrypoint=/bin/validate-from-dir.sh \
    --net=host --rm \
    -v /path/to/config-folder:/workspace \
    -e DATASOURCE_DIR=/workspace \
    vmware/kube-fluentd-operator:latest
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

```xml
<match $labels(app=foo)>
  @type relabel
  @label @blackhole
</match>

<match $labels(app=bar)>
  @type relabel
  @label @blackhole
</match>

<label @blackhole>
  <match **>
    @type null
  </match>
</label>

# at this point, foo and bar's logs are being handled in the @blackhole chain,
# the rest are still available for processing
<match **>
  @type ..
</match>
```

### I want to parse ingress-nginx access logs and send them to a different log aggregator

The ingress controller uses a format different than the plain Nginx. You can use this fragment to configure the namespace hosting the ingress-nginx controller:

```xml
<filter $labels(app=nginx-ingress, _container=nginx-ingress-controller)>
  @type parser
  key_name log
  reserve_data true
  <parse>
    @type regexp
    expression /(?<remote_addr>[^ ]*) - \[(?<proxy_protocol_addr>[^ ]*)\] - (?<remote_user>[^ ]*) \[(?<time>[^\]]*)\] "(?<method>\S+)(?: +(?<request>[^\"]*) +\S*)?" (?<code>[^ ]*) (?<size>[^ ]*) "(?<referer>[^\"]*)" "(?<agent>[^\"]*)" (?<request_length>[^ ]*) (?<request_time>[^ ]*) \[(?<proxy_upstream_name>[^ ]*)\] (?<upstream_addr>[^ ]*) (?<upstream_response_length>[^ ]*) (?<upstream_response_time>[^ ]*) (?<upstream_status>[^ ]*)/
    time_format %d/%b/%Y:%H:%M:%S %z
  </parse>
</filter>

<match **>
  # send the parsed access logs here
</match>
```

The above configuration assumes you're using the Helm charts for Nginx ingress. If not, make sure to the change the `app` and `_container` labels accordingly. Given the horrendous regex above, you really should be outputting access logs in json format and just specify `@type json`.

### I want to send logs to different sinks based on log contents

The `retag` plugin allows to split a log stream based on whether the contents of certain fields match the given regular expressions.

```xml
<match $labels(app=apache)>
  @type retag
  <rule>
    key message
    pattern ^ERR
    tag notifications.error
  </rule>
  <rule>
    key message
    pattern ^ERR
    invert true
    tag notifications.other
  </rule>
</match>

<match $tag(notifications.error)>
  # manage log stream with error severity
</match>

<match $tag(notifications.**)>
  # manage log stream with non-error severity
</match>
```

### I have my kubectl configured and my configmaps ready. I want to see the generated files before deploying the Helm chart

You need to run `make` like this:

```bash
make run-once
```

This will build the code, then `config-reloader` will connect to the K8S cluster, fetch the data and generate \*.conf files in the `./tmp` directory. If there are errors the namespaces will be annotated.

### I want to build a custom image with my own fluentd plugin

Use the `vmware/kube-fluentd-operator:TAG` as a base and do any modification as usual. If this plugin is not top-secret consider sending us a patch :)

### I run two clusters - in us-east-2 and eu-west-2. How to differentiate between them when pushing logs to a single location?

When deploying the daemonset using Helm, make sure to pass some metadata:

For the cluster in USA:

```bash
helm install ... \
  --set=meta.key=cluster_info \
  --set=meta.values.region=us-east-2
```

For the cluster in Europe:

```bash
helm install ... \
  --set=meta.key=cluster_info \
  --set=meta.values.region=eu-west-2
```

If you are using ELK you can easily get only the logs from Europe using `cluster_info.region: +eu-west-2`. In this example the metadata key is `cluster_info` but you can use any key you like.

### I don't want to annotate all my namespaces at all

It is possible to reduce configuration burden by using a default configmap name. The default value is `fluentd-config` - kube-fluentd-operator will read the configmap by that name if the namespace is not annotated.
If you don't like this default name or happen to use this configmap for other purposes then override the default with `--default-configmap=my-default`.

### How can I be sure to use a valid path for the .pos and .buf files

.pos files store the progress of the upload process and .buf are used for local buffering. Colliding .pos/.buf paths can lead to races in Fluentd. As such, `kube-fluentd-operator` tries hard to rewrite such path-based parameters in a predictable way. You only need to make sure they are unique for your namespace and `config-reloader` will take care to make them unique cluster-wide.

### I dont like the annotation name logging.csp.vmware.com/fluentd-configmap

Use `--annotation=acme.com/fancy-config` to use acme.com/fancy-config as annotation name. However, you'd also need to customize the Helm chart. Patches are welcome!

## Known Issues

Currently space-delimited tags are not supported. For example, instead of `<filter a b>`, you need to use `<filter a>` and `<filter b>`.
This limitation will be addressed in a later version.

## Releases

- [CHANGELOG.md](CHANGELOG.md).

## Resoures

- This plugin is used to provide kubernetes metadata https://github.com/fabric8io/fluent-plugin-kubernetes_metadata_filter
- This daemonset definition is used as a template: https://github.com/fluent/fluentd-kubernetes-daemonset/tree/master/docker-image/v0.12/debian-elasticsearch, however `kube-fluentd-operator` uses version 1.x version of fluentd and all the compatible plugin versions.
- This [Github issue](https://github.com/fabric8io/fluent-plugin-kubernetes_metadata_filter/issues/73) was the inspiration for the project. In particular it borrows the tag rewriting based on Kubernetes metadata to allow easier routing after that.

## Contributing

The kube-fluentd-operator project team welcomes contributions from the community. If you wish to contribute code and you have not
signed our contributor license agreement (CLA), our bot will update the issue when you open a Pull Request. For any
questions about the CLA process, please refer to our [FAQ](https://cla.vmware.com/faq). For more detailed information,
refer to [CONTRIBUTING.md](CONTRIBUTING.md).
