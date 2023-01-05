#!/bin/bash

set -x 

images=("webmng-apache-ubuntu" "webmng-nginx-ubuntu")

for image in "${images[@]}"; do
    docker rm -f $(docker ps -a -q --filter ancestor="$image") &> /dev/null
    docker rmi $(docker images -q "$image") &> /dev/null
done
