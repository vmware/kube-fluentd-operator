#!/usr/bin/env bash
set -e

# Copyright © 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: BSD-2-Clause

# roughly based on https://github.com/fluent/fluentd-kubernetes-daemonset/blob/master/docker-image/v0.12/debian-elasticsearch/entrypoint.sh
# the main difference is that a failsafe configuration is generated at boot time
# to prevent pod failing

main() {
  # Make sure the default configuration is under /fluentd/etc
  # it should be already copied in the Dockerfile
  DEFAULT_FLUENT_CONF=/fluentd/etc/${FLUENTD_CONF:-fluent.conf}
  echo "Using configuration: ${DEFAULT_FLUENT_CONF}"

  local retries=${RETRIES:-60}
  for attempt in $( seq 1 "$retries" ); do
    if [[ -s ${DEFAULT_FLUENT_CONF} ]]; then
      local ready=true
      break
    fi
    echo "Waiting for config file to become available: $attempt of $retries"
    sleep 5
  done
  if [[ "$ready" != "true" ]]; then
    return 1
  fi

  echo "Found configuration file: ${DEFAULT_FLUENT_CONF}"
  # shellcheck disable=SC2086
  exec fluentd -c "${DEFAULT_FLUENT_CONF}" -p /fluentd/plugins ${FLUENTD_OPT}
}

main
