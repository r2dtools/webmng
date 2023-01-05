#!/bin/bash

set -x 

images=("webmng-apache-ubuntu" "webmng-nginx-ubuntu" "webmng-apache-centos" "webmng-apache-almalinux")

for image in "${images[@]}"; do
    docker rm -f $(docker ps -a -q --filter ancestor="$image") &>/dev/null 
done
