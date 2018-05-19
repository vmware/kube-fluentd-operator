# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

FROM fluent/fluentd:v1.1.3-debian


# start with a valid empty file
COPY failsafe.conf /fluentd/failsafe.conf
COPY entrypoint.sh /fluentd/entrypoint.sh
ENTRYPOINT ["/fluentd/entrypoint.sh"]

# the below is based on https://github.com/fluent/fluentd-kubernetes-daemonset/tree/master/docker-image/v0.12/debian-elasticsearch
# the main difference is that all fluentd plugins are installed in a single image

USER root
WORKDIR /home/fluent

COPY plugins /fluentd/plugins
ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v0.12/debian-elasticsearch/plugins/parser_kubernetes.rb /fluentd/plugins

RUN buildDeps="sudo make gcc g++ libc-dev ruby-dev libffi-dev" \
     && apt-get update \
     && apt-get upgrade -y \
     && apt-get install \
     -y --no-install-recommends \
     $buildDeps \
    && echo 'gem: --no-document' >> /etc/gemrc \
    && fluent-gem install ffi \
    && fluent-gem install fluent-plugin-concat \
    && fluent-gem install fluent-plugin-elasticsearch \
    && fluent-gem install fluent-plugin-google-cloud \
    && fluent-gem install fluent-plugin-detect-exceptions \
    && fluent-gem install fluent-plugin-kubernetes \
    && fluent-gem install fluent-plugin-kubernetes_metadata_filter \
    && fluent-gem install fluent-plugin-logentries \
    && fluent-gem install fluent-plugin-loggly \
    && fluent-gem install fluent-plugin-logzio \
    && fluent-gem install fluent-plugin-papertrail \
    && fluent-gem install fluent-plugin-parser \
    && fluent-gem install fluent-plugin-record-modifier \
    && fluent-gem install fluent-plugin-record-reformer \
    && fluent-gem install fluent-plugin-redis \
    && fluent-gem install fluent-plugin-remote_syslog \
    && fluent-gem install fluent-plugin-rewrite-tag-filter \
    && fluent-gem install fluent-plugin-route \
    && fluent-gem install fluent-plugin-s3 \
    && fluent-gem install fluent-plugin-secure-forward \
    && fluent-gem install fluent-plugin-splunkhec \
    && fluent-gem install fluent-plugin-systemd \
    && fluent-gem install logfmt \
    && SUDO_FORCE_REMOVE=yes \
    apt-get purge -y --auto-remove \
                  -o APT::AutoRemove::RecommendsImportant=false \
                  $buildDeps \
 && rm -rf /var/lib/apt/lists/* \
    && gem sources --clear-all \
    && rm -rf /tmp/* /var/tmp/* /usr/lib/ruby/gems/*/cache/*.gem
