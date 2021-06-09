# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause
# Similar to https://github.com/drecom/docker-centos-ruby/blob/2.6.5-slim/Dockerfile

ARG RUBY_PATH=/usr/local
ARG RUBY_VERSION=2.7.3
ARG RUBYOPT='-W:no-deprecated -W:no-experimental'

FROM photon:4.0 AS rubybuild
ARG RUBY_PATH
ARG RUBY_VERSION
ARG RUBYOPT
RUN tdnf upgrade -y && \
    tdnf erase -y toybox && \
    tdnf install -y \
         bzip2 \
         shadow \
         wget \
         which \
         vim \
         less \
         tar \
         gzip \
         util-linux \
         sed \
         gcc \
         build-essential \
         zlib-devel \
         libedit \
         libedit-devel \
         gdbm \
         gdbm-devel \
         openssl-devel \
         systemd \
         net-tools \
         git

# Copy Gemfile.lock to pin versions further:
COPY basegems/Gemfile Gemfile
COPY basegems/Gemfile.lock Gemfile.lock

# Install the gems with bundler is better practice:
RUN git clone git://github.com/rbenv/ruby-build.git $RUBY_PATH/plugins/ruby-build \
    && $RUBY_PATH/plugins/ruby-build/install.sh \
    && ruby-build $RUBY_VERSION $RUBY_PATH/ \
    && gem update --system \
    && gem install bundler -v '>= 2.2.19' --default \
    && bundler install \
    && gem uninstall rake -v 13.0.3 \
    && gem uninstall bigdecimal

FROM photon:4.0
ARG RUBY_PATH
ARG RUBYOPT
ENV PATH $RUBY_PATH/bin:$PATH
COPY --from=rubybuild $RUBY_PATH $RUBY_PATH
# Not sure why this is needed: see https://github.com/fluent/fluentd-kubernetes-daemonset/blob/master/docker-image/v1.4/debian-elasticsearch/Dockerfile
ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1
# start with a valid empty file
COPY failsafe.conf /fluentd/failsafe.conf
# custom entrypoint
COPY entrypoint.sh /fluentd/entrypoint.sh

USER root

ENTRYPOINT ["/fluentd/entrypoint.sh"]
# Pin all fluentd plugin gem versions with Gemfile.lock here:
COPY Gemfile /fluentd/Gemfile
COPY Gemfile.lock /fluentd/Gemfile.lock
RUN mkdir -p /fluentd/log /fluentd/etc /fluentd/plugins /usr/local/bundle/bin/ \
  && tdnf erase -y toybox \
  && buildDeps="\
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
     openssl-devel" \
  && tdnf install -y $buildDeps util-linux systemd net-tools findutils \
  && wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_kubernetes.rb -P /fluentd/plugins \
  && wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_multiline_kubernetes.rb -P /fluentd/plugins \
  && echo 'gem: --no-document' >> /etc/gemrc \
  && bundle config silence_root_warning true \
  && cd /fluentd \
  && bundle install \
  && cd /fluentd \
  && git clone https://github.com/Cryptophobia/fluent-plugin-google-cloud.git fluent-plugin-google-cloud \
  && cd fluent-plugin-google-cloud \
  && gem build fluent-plugin-google-cloud.gemspec \
  && gem install fluent-plugin-google-cloud-*.gem \
  && cd /fluentd \
  && rm -rf /fluentd/fluent-plugin-google-cloud \
  && tdnf clean all \
  && gem sources --clear-all \
  && ln -s $(which fluentd) /usr/local/bundle/bin/fluentd \
  && tdnf remove -y $buildDeps

EXPOSE 24444 5140
COPY plugins /fluentd/plugins
