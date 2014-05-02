#!/bin/bash

set -e

GOPATH=$(pwd)/vendor/go:$GOPATH go run *.go $*
