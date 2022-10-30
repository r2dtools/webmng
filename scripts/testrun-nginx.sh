#!/bin/bash

set -e

service nginx start
go test "$@" ./internal/nginx/...
