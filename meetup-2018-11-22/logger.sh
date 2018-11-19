#!/bin/bash

ANIMAL="${ANIMAL:-some unknown animal}"
SOUND="${SOUND:-???}"


while true; do
  export now="$(date --rfc-3339=seconds)"
  export msg="I am $ANIMAL and I say '${SOUND}' #msgid$((var++))"
  export level="INFO"

  if [[ $(($RANDOM % 10)) == 0 ]]; then
    msg="Severe: $ANIMAL is unable to say '${SOUND}' #msgid$((var++))"
    level="ERROR"
  fi
  echo "$now [$level] $msg"

  if [[ "$FILE" != '' ]]; then
    echo '{ "timestamp": "$now", "level": "$level", "message": "$msg"}' | envsubst >> $FILE
  fi

  sleep 2
done
