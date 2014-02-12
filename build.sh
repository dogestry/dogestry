#!/bin/bash

set -e

d="sudo docker"

$d build -t dogestry .
id=$($d inspect dogestry | jq -r '.[0].container')
$d cp $id:dogestry .

BUCKET=ops-data-oregon.blakedev.com
REGION=us-west-2
aws s3 cp --acl public-read --region $REGION dogestry s3://$BUCKET/bin/dogestry

