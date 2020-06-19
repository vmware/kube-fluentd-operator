# Copyright © 2018 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause
# Similar to https://github.com/drecom/docker-centos-ruby/blob/2.6.5-slim/Dockerfile

ARG RUBY_PATH=/usr/local
ARG RUBY_VERSION=2.6.5

FROM photon:3.0 AS rubybuild
ARG RUBY_PATH
ARG RUBY_VERSION
RUN tdnf erase -y toybox && tdnf install -y bzip2 shadow wget which vim less tar gzip util-linux sed gcc build-essential zlib-devel libedit libedit-devel gdbm gdbm-devel openssl-devel systemd net-tools git

COPY basegems/Gemfile Gemfile
RUN git clone git://github.com/rbenv/ruby-build.git $RUBY_PATH/plugins/ruby-build \
    &&  $RUBY_PATH/plugins/ruby-build/install.sh \
    && ruby-build $RUBY_VERSION $RUBY_PATH/ \
    && gem install -g --file Gemfile \
    && gem uninstall rake -v 12.3.2 \
    && gem uninstall bigdecimal -v 1.4.1

FROM photon:3.0
ARG RUBY_PATH
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
# list gems here
COPY Gemfile /fluentd
RUN mkdir -p /fluentd/log /fluentd/etc /fluentd/plugins /usr/local/bundle/bin/ \
	&& tdnf erase -y toybox \
	&& buildDeps="\
	gmp-devel libffi-devel bzip2 shadow wget which vim less tar gzip sed gcc build-essential zlib-devel libedit libedit-devel gdbm gdbm-devel openssl-devel" \
	&& tdnf install -y $buildDeps util-linux systemd net-tools\
	&& wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_kubernetes.rb -P /fluentd/plugins \
	&& wget https://raw.githubusercontent.com/fluent/fluentd-kubernetes-daemonset/master/docker-image/v1.4/debian-elasticsearch/plugins/parser_multiline_kubernetes.rb -P /fluentd/plugins \
	&& echo 'gem: --no-document' >> /etc/gemrc \
	&& bundle config silence_root_warning true \
	&& wget https://github.com/vmware/fluent-plugin-vmware-log-intelligence/releases/download/v2.0.0/fluent-plugin-vmware-log-intelligence-2.0.0.gem \
	&& bundle install --gemfile=/fluentd/Gemfile \
	&& fluent-gem install --local fluent-plugin-vmware-log-intelligence-2.0.0.gem \
	&& tdnf clean all \
	&& gem sources --clear-all \
	&& ln -s $(which fluentd) /usr/local/bundle/bin/fluentd \
	&& tdnf remove -y $buildDeps \
	&& rm -rf /tmp/* /var/tmp/* /usr/lib/ruby/gems/*/cache/*.gem /fluent-plugin-vmware-log-intelligence-2.0.0.gem

EXPOSE 24444 5140
COPY plugins /fluentd/plugins
