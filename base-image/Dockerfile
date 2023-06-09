# Copyright © 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause
# Similar to https://github.com/drecom/docker-centos-ruby/blob/2.6.5-slim/Dockerfile

ARG RVM_PATH=/usr/local/rvm
ARG RUBY_VERSION=ruby-3.1.4
ARG RUBY_PATH=/usr/local/rvm/rubies/$RUBY_VERSION
ARG RUBYOPT='-W:no-deprecated -W:no-experimental'

FROM photon:3.0 AS rubybuild
ARG RVM_PATH
ARG RUBY_PATH
ARG RUBY_VERSION
ARG RUBYOPT
RUN tdnf clean all && \
    tdnf upgrade -y && \
    tdnf erase -y toybox && \
    tdnf install -y \
         findutils \
         procps-ng \
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
         git \
         gpg && \
    tdnf clean all

# Copy Gemfile.lock to pin versions further:
COPY basegems/Gemfile Gemfile
COPY basegems/Gemfile.lock Gemfile.lock

SHELL [ "/bin/bash", "-l", "-c" ]

# Install the gems with bundler is better practice:
RUN curl -sSL https://rvm.io/mpapis.asc | gpg --import \
    && curl -sSL https://rvm.io/pkuczynski.asc | gpg --import \
    && curl -sSL https://get.rvm.io | bash -s stable \
    && source /etc/profile.d/rvm.sh \
    && rvm autolibs disable \
    && rvm requirements \
    && rvm install --disable-binary $RUBY_VERSION --default \
    && gem update --system --no-document \
    && gem install bundler -v '>= 2.3.26' --default --no-document \
    && bundler install

FROM photon:3.0
ARG RUBY_PATH
ARG RUBYOPT
ENV PATH $RUBY_PATH/bin:$PATH
COPY --from=rubybuild $RUBY_PATH $RUBY_PATH
# Not sure why this is needed: see https://github.com/fluent/fluentd-kubernetes-daemonset/blob/master/docker-image/v1.13/debian-elasticsearch7/Dockerfile
# skip runtime bundler installation
ENV FLUENTD_DISABLE_BUNDLER_INJECTION 1

COPY failsafe.conf entrypoint.sh Gemfile Gemfile.lock /fluentd/

USER root

ENTRYPOINT ["/fluentd/entrypoint.sh"]

RUN mkdir -p /fluentd/log /fluentd/etc /fluentd/plugins /usr/local/bundle/bin/ \
  && tdnf clean all \
  && tdnf upgrade -y \
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
  && wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.15/debian-elasticsearch7/plugins/parser_kubernetes.rb -P /fluentd/plugins \
  && wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.15/debian-elasticsearch7/plugins/parser_multiline_kubernetes.rb -P /fluentd/plugins \
  && echo 'gem: --no-document' >> /etc/gemrc \
  && bundle config silence_root_warning true \
  && cd /fluentd \
  && bundle install \
  && cd /fluentd \
  && git clone https://github.com/slimm609/fluent-plugin-detect-exceptions.git fluent-plugin-detect-exceptions \
  && cd fluent-plugin-detect-exceptions \
  && gem build fluent-plugin-detect-exceptions.gemspec \
  && gem install fluent-plugin-detect-exceptions-*.gem \
  && rm -rf /fluentd/fluent-plugin-detect-exceptions \
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
  && rm -rf /fluentd/fluent-plugin-loggly \
  && wget https://github.com/jemalloc/jemalloc/releases/download/5.3.0/jemalloc-5.3.0.tar.bz2 -P /tmp \
  && tar -C /tmp/ -xjvf /tmp/jemalloc-5.3.0.tar.bz2 \
  && cd /tmp/jemalloc-5.3.0 \
  && ./configure && make \
  && mv -v lib/libjemalloc.so* /usr/lib \
  && rm -rf /tmp/jemalloc-5.3.0 /tmp/jemalloc-5.3.0.tar.bz2 \
  && cd /fluentd \
  && gem sources --clear-all \
  && ln -s $(which fluentd) /usr/local/bundle/bin/fluentd \
  && tdnf remove -y $buildDeps \
  && tdnf clean all \
  && gem uninstall google-protobuf --version 3.17.3 --force \
  && gem uninstall bundler \
  && gem cleanup \
  && rm -rf $RUBY_PATH/lib/ruby/gems/2.7.0/cache $RUBY_PATH/lib/ruby/gems/2.7.0/doc/ /usr/share/doc /root/.bundle/cache

# Make sure fluentd picks jemalloc 5.3.0 lib as default
ENV LD_PRELOAD="/usr/lib/libjemalloc.so"

EXPOSE 24444 5140
COPY plugins /fluentd/plugins
