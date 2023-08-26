# Copyright © 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause
# Similar to https://github.com/drecom/docker-centos-ruby/blob/2.6.5-slim/Dockerfile


FROM golang:1.19 as builder

WORKDIR /go/src/github.com/vmware/kube-fluentd-operator/config-reloader
COPY config-reloader .
COPY Makefile .

# Speed up local builds where vendor is populated
ARG VERSION
# Good to have another test run as this container is different than lint container
RUN make in-docker-test
RUN make build VERSION=$VERSION

FROM photon:4.0 

ARG RVM_PATH=/usr/local/rvm
ARG RUBY_VERSION=ruby-3.1.4
ARG RUBY_PATH=/usr/local/rvm/rubies/$RUBY_VERSION
ARG RUBYOPT='-W:no-deprecated -W:no-experimental'

ENV PATH $RUBY_PATH/bin:$PATH
ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1
ENV BUILDDEPS="\
      gmp-devel \
      libffi-devel \
      bzip2 \
      shadow \
      which \
      wget \
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
      openssl-devel \
      gpg"

RUN tdnf clean all && \
    tdnf upgrade -y && \
    tdnf erase -y toybox && \
    tdnf install -y \
         findutils \
         procps-ng \
         util-linux \
         systemd \
         net-tools && \
    tdnf clean all

SHELL [ "/bin/bash", "-l", "-c" ]

COPY image/failsafe.conf image/entrypoint.sh image/Gemfile image/Gemfile.lock /fluentd/

# Install the gems with bundler is better practice
# We need to keep this as a single layer because of the builddeps
# if we split between multiple steps, we need up with the lots of extra files between layers
RUN tdnf install -y $BUILDDEPS \
  && curl -sSL https://rvm.io/mpapis.asc | gpg --import \
  && curl -sSL https://rvm.io/pkuczynski.asc | gpg --import \
  && curl -sSL https://get.rvm.io | bash -s stable \
  && source /etc/profile.d/rvm.sh \
  && rvm autolibs disable \
  && rvm requirements \
  && rvm install --disable-binary $RUBY_VERSION --default \
  && gem update --system --no-document \
  && gem install bundler -v '>= 2.4.15' --default --no-document \
  && rm -rf $RVM_PATH/src $RVM_PATH/examples $RVM_PATH/docs $RVM_PATH/archives \
    $RUBY_PATH/lib/ruby/gems/3.*/cache $RUBY_PATH/lib/ruby/gems/3.*/doc/ \
    /usr/share/doc /root/.bundle/cache \
  && rvm cleanup all \  
  && gem sources --clear-all \
  && gem cleanup \ 
  && tdnf remove -y $BUILDDEPS \
  && tdnf clean all

RUN tdnf install -y $BUILDDEPS \
  && mkdir -p /fluentd/log /fluentd/etc /fluentd/plugins /usr/local/bundle/bin/ \
  && echo 'gem: --no-document' >> /etc/gemrc \
  && bundle config silence_root_warning true \
  && cd /fluentd \
  && bundle install \
  && cd /fluentd \
  && gem specific_install https://github.com/javiercri/fluent-plugin-google-cloud.git \
  && cd /fluentd \
  && gem sources --clear-all \
  && ln -s $(which fluentd) /usr/local/bundle/bin/fluentd \
  && gem cleanup \
  ## Install jemalloc
  && curl -sLo /tmp/jemalloc-5.3.0.tar.bz2 https://github.com/jemalloc/jemalloc/releases/download/5.3.0/jemalloc-5.3.0.tar.bz2 \
  && tar -C /tmp/ -xjvf /tmp/jemalloc-5.3.0.tar.bz2 \
  && cd /tmp/jemalloc-5.3.0 \
  && ./configure && make \
  && mv -v lib/libjemalloc.so* /usr/lib \
  && rm -rf /tmp/* \
  # cleanup build deps
  && tdnf remove -y $BUILDDEPS \
  && tdnf clean all

COPY image/plugins /fluentd/plugins

COPY config-reloader/templates /templates
COPY config-reloader/validate-from-dir.sh /bin/validate-from-dir.sh
COPY --from=builder /go/src/github.com/vmware/kube-fluentd-operator/config-reloader/config-reloader /bin/config-reloader

# Make sure fluentd picks jemalloc 5.3.0 lib as default
ENV LD_PRELOAD="/usr/lib/libjemalloc.so"

EXPOSE 24444 5140

USER root

ENTRYPOINT ["/fluentd/entrypoint.sh"]