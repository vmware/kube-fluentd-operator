# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

FROM fluent/fluentd:v1.7.4-debian-1.0

# rever user fluent from parent
USER root

# Not sure why this is needed: see https://github.com/fluent/fluentd-kubernetes-daemonset/blob/master/docker-image/v1.4/debian-elasticsearch/Dockerfile
ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1

# start with a valid empty file
COPY failsafe.conf /fluentd/failsafe.conf

# custom entrypoint
COPY entrypoint.sh /fluentd/entrypoint.sh

ENTRYPOINT ["tini", "--", "/fluentd/entrypoint.sh"]

# copy k8s config
ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_kubernetes.rb /fluentd/plugins
ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_multiline_kubernetes.rb /fluentd/plugins

# list gems here
COPY Gemfile /fluentd

ENV DEBIAN_FRONTEND=noninteractive
RUN buildDeps="make gcc g++ libc-dev ruby-dev libffi-dev wget" \
    && apt-get update \
    && apt-get upgrade -y \
    && apt-get install -y --no-install-recommends \
      $buildDeps net-tools \
    && echo 'gem: --no-document' >> /etc/gemrc \
    && gem install bundler --version 1.16.2 \
    && bundle config silence_root_warning true \
    && bundle install --gemfile=/fluentd/Gemfile \
    && wget https://github.com/vmware/fluent-plugin-vmware-log-intelligence/releases/download/v2.0.0/fluent-plugin-vmware-log-intelligence-2.0.0.gem \
    && fluent-gem install --local fluent-plugin-vmware-log-intelligence-2.0.0.gem \
    && rm fluent-plugin-vmware-log-intelligence-2.0.0.gem \
    && SUDO_FORCE_REMOVE=yes apt-get purge -y --auto-remove -o APT::AutoRemove::RecommendsImportant=false \
        $buildDeps \
    && rm -rf /var/lib/apt/lists/* \
    && gem sources --clear-all \
    && rm -rf /tmp/* /var/tmp/* /usr/lib/ruby/gems/*/cache/*.gem

COPY plugins /fluentd/plugins
