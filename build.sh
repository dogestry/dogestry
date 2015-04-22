#!/bin/bash

set -e

[[ -z $DOCKER ]] && DOCKER=docker

# don't rm intermediate containers... we want them!
$DOCKER build --rm=false -t dogestry .
id=$($DOCKER inspect -f '{{ .container }}' dogestry)
$DOCKER cp $id:dogestry .

if [ -f "./push.sh" ]; then
  ./push.sh
fi
