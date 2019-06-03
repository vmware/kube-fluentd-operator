# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

FROM fluent/fluentd:v1.2.6-debian


# start with a valid empty file
COPY failsafe.conf /fluentd/failsafe.conf
COPY entrypoint.sh /fluentd/entrypoint.sh
ENTRYPOINT ["/fluentd/entrypoint.sh"]

# https://github.com/fluent/fluentd-kubernetes-daemonset/blob/master/docker-image/v1.1/debian-elasticsearch/Dockerfile
# the main difference is that all fluentd plugins are installed in a single image

USER root
WORKDIR /home/fluent

ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.2/debian-elasticsearch/plugins/parser_kubernetes.rb /fluentd/plugins
ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.2/debian-elasticsearch/plugins/parser_multiline_kubernetes.rb /fluentd/plugins

RUN buildDeps="sudo make gcc g++ libc-dev ruby-dev libffi-dev wget" \
     && apt-get update \
     && apt-get upgrade -y \
     && apt-get install \
     -y --no-install-recommends \
     $buildDeps libjemalloc1 \
    && echo 'gem: --no-document' >> /etc/gemrc \
    && fluent-gem install ffi \
    && fluent-gem install fluent-plugin-amqp \
    && fluent-gem install fluent-plugin-cloudwatch-logs \
    && fluent-gem install fluent-plugin-concat \
    && fluent-gem install fluent-plugin-detect-exceptions \
    && fluent-gem install fluent-plugin-elasticsearch \
    && fluent-gem install fluent-plugin-google-cloud \
    && fluent-gem install fluent-plugin-grok-parser \
    && fluent-gem install fluent-plugin-kafka \
    && fluent-gem install fluent-plugin-kinesis \
    && fluent-gem install fluent-plugin-kubernetes \
    && fluent-gem install fluent-plugin-kubernetes_metadata_filter \
    && fluent-gem install fluent-plugin-logentries \
    && fluent-gem install fluent-plugin-loggly \
    && fluent-gem install fluent-plugin-logzio \
    && fluent-gem install fluent-plugin-mail \
    && fluent-gem install fluent-plugin-mongo \
    && fluent-gem install fluent-plugin-out-http-ext \
    && fluent-gem install fluent-plugin-papertrail \
    && fluent-gem install fluent-plugin-prometheus \
    && fluent-gem install fluent-plugin-record-modifier \
    && fluent-gem install fluent-plugin-record-reformer \
    && fluent-gem install fluent-plugin-redis \
    && fluent-gem install fluent-plugin-remote_syslog \
    && fluent-gem install fluent-plugin-rewrite-tag-filter \
    && fluent-gem install fluent-plugin-route \
    && fluent-gem install fluent-plugin-s3 \
    && fluent-gem install fluent-plugin-scribe \
    && fluent-gem install fluent-plugin-secure-forward \
    && fluent-gem install fluent-plugin-splunkhec \
    && fluent-gem install fluent-plugin-sumologic_output \
    && fluent-gem install fluent-plugin-systemd \
    && fluent-gem install fluent-plugin-vertica \
    && fluent-gem install fluent-plugin-verticajson \
    && fluent-gem install fluent-plugin-kubernetes_sumologic \
    && fluent-gem install logfmt \
    && wget https://github.com/vmware/fluent-plugin-vmware-log-intelligence/releases/download/v1.0.0/fluent-plugin-vmware-log-intelligence-1.0.0.gem \
    && fluent-gem install --local fluent-plugin-vmware-log-intelligence-1.0.0.gem \
    && rm fluent-plugin-vmware-log-intelligence-1.0.0.gem \
    && SUDO_FORCE_REMOVE=yes \
    apt-get purge -y --auto-remove \
                  -o APT::AutoRemove::RecommendsImportant=false \
                  $buildDeps \
 && rm -rf /var/lib/apt/lists/* \
    && gem sources --clear-all \
    && rm -rf /tmp/* /var/tmp/* /usr/lib/ruby/gems/*/cache/*.gem

# See https://packages.debian.org/stretch/amd64/libjemalloc1/filelist
ENV LD_PRELOAD="/usr/lib/x86_64-linux-gnu/libjemalloc.so.1"

COPY plugins /fluentd/plugins
