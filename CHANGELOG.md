# CHANGELOG

## [1.12.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.12.0)

* Build kube-fluentd-operator on photon (#115) (@dimalbaby)

* Plugins: Add `fluent-plugin-vmware-loginsight` (@naveens)

* Plugins: Add `fluent-plugin-grafana-loki` (@SerialVelocity)

* Core: Cleanup unused namespace configuration files (#110) (@tommasopozzetti)

* Core: Fix labels not being rewritten in copy plugin (#86) (@tommasopozzetti)

* Core: Fix $label processor ignoring errors and not accepting "/" (#93) (@tommasopozzetti)

* Core: Fix namespace fluentd status annotation update (#92) (@tommasopozzetti)

* Core: Fix slow CRI-O log parsing (#91) (@jonasrutishauser)

* Core: Introduce a more efficient event-triggered reload system (#89)  (@tommasopozzetti)

* Core: Add new processor to allow controlled tag rewriting (#88) (@tommasopozzetti)

* Helm: improve Helm chart and bump version to 0.3.3 (#95) (@cablespaghetti)

* Fluentd: upgrade fluentd to 1.9.1

## [1.11.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.11.0)

* Plugins: Add `fluent-plugin-verticajson` again (@skalickym)

* Plugins: Add `fluent-plugin-uri-parser` (@rjackson90)

* Plugins: Add `fluent-plugin-out-http`

* Plugins: update to latest compatible versions

* Core: Efficient Kubernetes API Updates using Informers (#75) (@rjackson90)

* Fluend: Add support for CRI-O format (#80) (@jonasrutishauser)

## [1.10.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.10.0)

* Fluentd: Upgrade to 1.5.2

* Fluentd: Use libjemalloc from base image improving multi-platform build (@james-crowley)

* Plugins: Add `fluent-plugin-datadog` (@chirauki)

* Plugins: Add `fluent-plugin-multi-format-parser`

* Plugins: Add `fluent-plugin-json-in-json-2`

* Plugins: Add `fluent-plugin-gelf-hs`

* Plugins: Add `fluent-plugin-azure-loganalytics` (@zachomedia)

* Plugins: Upgrade LINT plugin to version 2.0.0

* Plugins: Remove vertica plugins as they fail to build with 1.5.2

* Core: `<plugin>`'s content is expanded correctly (#62) (@)

* Core: Allow the use of virtual `<plugin>` in store directive (#71) (@zachomedia)

* Core: Add subPath support to mounted-file (#63) (@jonasrutishauser)

## [1.9.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.9.0)

* Fluentd: BREAKING CHANGE: the plugin `fluent-plugin-parser` is not installed anymore. Instead the built-in `@type parser`
  is used and it has a different syntax (https://github.com/tagomoris/fluent-plugin-parser)

* Fluentd: Updated K8S integration to latest recommended for Fluentd 1.2.x

* Plugins: refresh all plugins to latest versions as of the release date.

* Plugins: add plugin `cloudwatch-logs` (@conradj87)

* Helm: improve Helm chart and bump version to 0.3.1 (@PabloCastellano)

## [1.8.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.8.0)

* Fluentd: replace fluent-plugin-amqp plugin with fluent-plugin-amqp2

* Fluentd: update fluentd to 1.2.6

* Fluentd: add plugin `fluent-plugin-vertica` to base image (@rhatlapa)

* Core configmaps per namespace support (#31) (@coufalja)

* Helm: chart updated to 0.3.0 adding support for multiple configmaps per namespace

## [1.7.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.7.0)

* Fluentd: add plugin `extract` to base image - a simple filter plugin to enrich log events based on regex and templates

* Fluentd: add plugin `fluent-plugin-amqp2` to base image (@CoufalJa)

* Fluentd: add plugin `fluent-plugin-grok-parser` to base image (@CoufalJa)

* Core: handle gracefully a missing `kubernetes.labels` field (#23)

* Core: Add `strict true|false` option to the logfmt parser plugin (#27)

* Core: Add `add_labels` to  `@type mounted_file` plug-in (#26)

* Core: Fix `@type mounted_file` for multi-container pods producing log files (#29)

* Helm: `podAnnotations` lets you annotate the daemonset pods (@cw-sakamoto) (#25)

## [1.6.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.6.0)

* Core: plugin macros (#19). This feature lets you reuse output plugin definitions.

* Fluentd: add `fluent-plugin-kafka` to base image (@dimitrovvlado)

* Fluentd: add `fluent-plugin-mail` to base image

* Fluentd: add `fluent-plugin-mongo` to base image

* Fluentd: add `fluent-plugin-scribe` to base image

* Fluentd: add `fluent-plugin-sumologic_output` to base image

## [1.5.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.5.0)

* Core: extreme validation (#15): namespace configs are run in sandbox to catch more errors

* Helm: export a K8S\_NODE\_NAME var to the fluentd container (@databus23)

* Helm: `extraEnv` lets you inject env vars into fluentd

## [1.4.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.4.0)

* Fluentd: add `fluent-plugin-kinesis` to base image (@anton-107)

## [1.3.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.3.0)

* Fluentd: add `fluent-plugin-detect-exceptions` to base image

* Fluentd: add `fluent-plugin-out-http-ext` to base image

* Core: add plugin `truncating_remote_syslog` which truncates the tag to 32 bytes as per RFC 5424

* Core: support transparent multi-line stacktrace collapsing

## [1.2.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.2.0)

* Core: Share log streams between namespaces

* Helm: Mount secrets/configmaps (tls mostly) ala elasticsearch (@sneko)

* Fluentd: Update base-image to fluentd-1.1.3-debian

* Fluentd: Include Splunk plugin into base-image (@mhulscher)

* Fix(Helm): properly set the `resources` field of the reloader container. Setting them had no effect until now (@sneko)

## [1.1.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.1.0)

* Core: ingest log files from a container filesystem

* Core: limit impact of log-router to a set of namespaces using `--namespaces`

* Helm: add new property `kubeletRoot`

* Helm: add new property `namespaces[]`:

## [1.0.1](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.0.1)

* Fluentd: install plugin `fluent-plugin-concat` in Fluentd

* Core: support for default configmap name using `--default-configmap`

## [1.0.0](https://github.com/vmware/kube-fluentd-operator/releases/tag/v1.0.0)

* Initial version
