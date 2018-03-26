#!/bin/bash

# this is used for local test as fluentd may not be installed
# no validation is performed just the invocation parameters are printed

if [[ $1 == "--version" ]]; then
  echo "fake-fluentd 1.0"
  exit 0
fi

echo
echo Invoked with "$@"
echo __________________
if [[ "$5" != "" ]]; then
  cat $5
else 
  echo "<<<nothing provided as file to validate>>>"
fi
echo __________________

# for unit tests: if the input contains #ERROR, exit with 1
if grep 'ERROR' < "$5" ; then
  exit 1
fi

exit 0
