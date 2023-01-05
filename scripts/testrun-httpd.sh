#!/bin/bash

set -e

/usr/sbin/httpd -k start
go test "$@" ./internal/apache/...
