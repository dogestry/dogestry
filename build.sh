#!/bin/bash

set -e

d="sudo docker"

$d build -t dogestry .
id=$($d inspect dogestry | jq -r '.[0].container')
$d cp $id:dogestry .

md=$(curl http://169.254.169.254/latest/meta-data/iam/security-credentials//buildbox-staging)
access_key_id=$(echo $md | jq -r .AccessKeyId)
secret_key=$(echo $md | jq -r .SecretAccessKey)

S3CFG=.s3cfg
cat <<EOC > $S3CFG
[default]
aws_access_key=$access_key_id
aws_secret_key=$secret_key
region=us-west-2
EOC
trap EXIT "rm $S3CFG"

chmod 0600 $S3CFG

BUCKET=ops-data-oregon.blakedev.com
AWS_CONFIG_FILE=$S3CFG aws s3 cp --acl public-read dogestry s3://$BUCKET/bin/dogestry

