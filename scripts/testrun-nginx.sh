#!/bin/bash

set -e

go test "$@" ./internal/nginx/...
