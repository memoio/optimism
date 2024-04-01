#!/bin/bash
set -eux

DOCKER_COMPOSE_PATH=./ops-bedrock/docker-compose.yml

if [ $# == 0 ]; then
    # default
    DA_RPC_ADDR="http://127.0.0.1:26650"
else
    # specific rpc address
    DA_RPC_ADDR=$1
fi

sed -i "s|DA_RPC: \".*\"|DA_RPC: \"$DA_RPC_ADDR\"|g" $DOCKER_COMPOSE_PATH
