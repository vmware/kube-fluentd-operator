# Copyright © 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause
# Similar to https://github.com/drecom/docker-centos-ruby/blob/2.6.5-slim/Dockerfile

FROM photon:3.0 

ARG RVM_PATH=/usr/local/rvm
ARG RUBY_VERSION=ruby-3.1.4
ARG RUBY_PATH=/usr/local/rvm/rubies/$RUBY_VERSION
ARG RUBYOPT='-W:no-deprecated -W:no-experimental'

RUN tdnf clean all && \
    tdnf upgrade -y && \
    tdnf erase -y toybox && \
    tdnf install -y \
         findutils \
         procps-ng \
         util-linux \
         systemd \
         net-tools \
         gpg && \
    buildDeps="\
      gmp-devel \
      libffi-devel \
      bzip2 \
      shadow \
      wget \
      which \
      vim \
      git \
      less \
      tar \
      gzip \
      sed \
      gcc \
      build-essential \
      zlib-devel \
      libedit \
      libedit-devel \
      gdbm \
      gdbm-devel \
      openssl-devel" && \
    tdnf install -y $buildDeps && \
    tdnf clean all

SHELL [ "/bin/bash", "-l", "-c" ]

COPY failsafe.conf entrypoint.sh Gemfile Gemfile.lock /fluentd/

# Install the gems with bundler is better practice:
RUN curl -sSL https://rvm.io/mpapis.asc | gpg --import \
    && curl -sSL https://rvm.io/pkuczynski.asc | gpg --import \
    && curl -sSL https://get.rvm.io | bash -s stable \
    && source /etc/profile.d/rvm.sh \
    && rvm autolibs disable \
    && rvm requirements \
    && rvm install --disable-binary $RUBY_VERSION --default \
    && gem update --system --no-document \
    && gem install bundler -v '>= 2.4.13' --default --no-document

ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1

## Install dependencies which are not up to date and we need a fork or local download
RUN mkdir -p /fluentd/log /fluentd/etc /fluentd/plugins /usr/local/bundle/bin/ \
  && wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.15/debian-elasticsearch7/plugins/parser_kubernetes.rb -P /fluentd/plugins \
  && wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.15/debian-elasticsearch7/plugins/parser_multiline_kubernetes.rb -P /fluentd/plugins \
  && echo 'gem: --no-document' >> /etc/gemrc \
  && bundle config silence_root_warning true \
  && cd /fluentd \
  && bundle install \
  && cd /fluentd \
  && git clone https://github.com/javiercri/fluent-plugin-google-cloud.git fluent-plugin-google-cloud \
  && cd fluent-plugin-google-cloud \
  && gem build fluent-plugin-google-cloud.gemspec \
  && gem install fluent-plugin-google-cloud-*.gem \
  && rm -rf /fluentd/fluent-plugin-google-cloud \
  && cd /fluentd \
  && git clone https://github.com/slimm609/fluent-plugin-loggly.git fluent-plugin-loggly \
  && cd fluent-plugin-loggly \
  && gem build fluent-plugin-loggly.gemspec \
  && gem install fluent-plugin-loggly-*.gem \
  && rm -rf /fluentd/fluent-plugin-loggly

COPY plugins /fluentd/plugins

## Install jemalloc
RUN wget https://github.com/jemalloc/jemalloc/releases/download/5.3.0/jemalloc-5.3.0.tar.bz2 -P /tmp \
  && tar -C /tmp/ -xjvf /tmp/jemalloc-5.3.0.tar.bz2 \
  && cd /tmp/jemalloc-5.3.0 \
  && ./configure && make \
  && mv -v lib/libjemalloc.so* /usr/lib \
  && rm -rf /tmp/jemalloc-5.3.0 /tmp/jemalloc-5.3.0.tar.bz2
# Make sure fluentd picks jemalloc 5.3.0 lib as default
ENV LD_PRELOAD="/usr/lib/libjemalloc.so"

## Remove all fluentd build deps and non needit configs
RUN cd /fluentd \
  && gem sources --clear-all \
  && ln -s $(which fluentd) /usr/local/bundle/bin/fluentd \
  && tdnf remove -y $buildDeps \
  && tdnf clean all \
  && gem uninstall bundler \
  && gem cleanup \
  && rm -rf $RUBY_PATH/lib/ruby/gems/3.1.4/cache $RUBY_PATH/lib/ruby/gems/3.1.4/doc/ /usr/share/doc /root/.bundle/cache

EXPOSE 24444 5140

USER root

ENTRYPOINT ["/fluentd/entrypoint.sh"]