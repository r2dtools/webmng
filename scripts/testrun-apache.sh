#!/bin/bash

set -e

apache2ctl -t
service apache2 restart
go test "$@" ./...
