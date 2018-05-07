# CHANGELOG

## 1.2.0 (Unreleased)

* Feature: share log streams between namespaces

* Fix(Helm): propery set the resources field of the reloader container. Setting them had no effect until now.

## 1.1.0

* Feature: ingest log files from a container filesystem

* Feature: limit impact of log-router to a set of namespaces using `--namespaces`

* Helm: add new property `kubeletRoot`

* Helm: add new property `namespaces[]`:

## 1.0.1

* install plugin `fluent-plugin-concat` in Fluentd

* support for default configmap name using `--default-configmap`

## 1.0.0

* initial version
