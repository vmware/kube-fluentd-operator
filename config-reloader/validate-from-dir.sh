#!/bin/bash

dest=$(mktemp -d)

config-reloader \
  --interval 0 \
  --log-level=debug \
  --output-dir=${dest} \
  --templates-dir=/templates \
  --datasource=fs \
  --exec-timeout=5 \
  --fs-dir=${DATASOURCE_DIR} \
  --fluentd-binary "/usr/local/bundle/bin/fluentd -p /fluentd/plugins"

rc=0
if [ -z "$(ls -A ${dest})" ]; then
  echo -n "*** Error destination folder is empty $dest"
  exit 1
fi
for st in $(find ${dest} -iname *.status); do
  ns=$(basename ${st})
  ns=${ns%%.status}
  ns=${ns#ns-}
  echo -n "*** Error for namespace '$ns': "
  cat "${st}"
  rc=1
done

exit $rc
