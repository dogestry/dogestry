#!/bin/bash

set -e

d="sudo docker"

# don't rm intermediate containers... we want them!
$d build --rm=false -t dogestry .
id=$($d inspect -f '{{ .container }}' dogestry)
$d cp $id:dogestry .

if [ -f "./push.sh" ]; then
  ./push.sh
fi
