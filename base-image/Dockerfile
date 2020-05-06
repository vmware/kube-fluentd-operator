# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

FROM photon:3.0

# rever user fluent from parent
USER root

# Not sure why this is needed: see https://github.com/fluent/fluentd-kubernetes-daemonset/blob/master/docker-image/v1.4/debian-elasticsearch/Dockerfile
ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1

# start with a valid empty file
COPY failsafe.conf /fluentd/failsafe.conf

# custom entrypoint
COPY entrypoint.sh /fluentd/entrypoint.sh

# list gems here
COPY Gemfile /fluentd

ENTRYPOINT ["/fluentd/entrypoint.sh"]

RUN tdnf update -y && tdnf upgrade -y && tdnf erase -y toybox
RUN tdnf install -y shadow wget which vim less tar gzip util-linux sed gcc build-essential zlib1g-dev zlib-devel libedit libedit-devel gdbm gdbm-devel openssl-devel systemd net-tools

RUN curl -o ruby.tgz https://cache.ruby-lang.org/pub/ruby/2.6/ruby-2.6.5.tar.gz \
  && tar xvzf ruby.tgz \
  && cd ruby-2.6.5 \
  && ./configure && make && make install

COPY basegems/Gemfile Gemfile
RUN gem install -g --file Gemfile && mv Gemfile Gemfile.base

ENV FLUENTD_CONF=""
ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1
ENV PATH /fluentd/vendor/bundle/ruby/2.6.0/bin:$PATH

RUN mkdir -p /fluentd/log /fluentd/etc /fluentd/plugins
COPY Gemfile /fluentd

RUN buildDeps="\
    gmp-devel libffi-devel" \
    && tdnf install -y $buildDeps \
    && echo 'gem: --no-document' >> /etc/gemrc \
    && bundle config silence_root_warning true \
    && wget https://github.com/vmware/fluent-plugin-vmware-log-intelligence/releases/download/v2.0.0/fluent-plugin-vmware-log-intelligence-2.0.0.gem \
    && bundle install --gemfile=/fluentd/Gemfile \
    && fluent-gem install --local fluent-plugin-vmware-log-intelligence-2.0.0.gem \
    && tdnf clean all \
    && tdnf remove -y $buildDeps \
    && gem sources --clear-all \
    && rm -rf /tmp/* /var/tmp/* /usr/lib/ruby/gems/*/cache/*.gem /fluent-plugin-vmware-log-intelligence-2.0.0.gem ruby.tgz ruby-2.6.5

EXPOSE 24444 5140


# copy k8s config
ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_kubernetes.rb /fluentd/plugins
ADD https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_multiline_kubernetes.rb /fluentd/plugins

COPY plugins /fluentd/plugins
RUN mkdir -p /usr/local/bundle/bin/ && ln -s $(which fluentd) /usr/local/bundle/bin/fluentd
RUN gem uninstall rake -v 12.3.2
