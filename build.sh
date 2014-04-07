#!/bin/bash

set -e

d="sudo docker"

$d build -t dogestry .
id=$($d inspect -f '{{ .container }}' dogestry)
$d cp $id:dogestry .

if [ -f "./push.sh" ]; then
  ./push.sh
fi
