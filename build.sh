#!/bin/bash

set -e

d="sudo docker"

$d build -t dogestry .
id=$($d inspect dogestry | jq -r '.[0].container')
$d cp $id:dogestry .
