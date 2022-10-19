#!/bin/bash

set -x 

docker rm -f $(docker ps -a -q --filter ancestor="webmng-test:latest") &> /dev/null
docker rmi $(docker images -q webmng-test) &> /dev/null
