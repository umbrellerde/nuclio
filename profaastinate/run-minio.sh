#!/usr/bin/env bash

docker run \
   -p 9000:9000 \
   -p 9090:9090 \
   --name minio \
   --rm \
   -it \
   --network profaastinate \
   -e "MINIO_ROOT_USER=minioadmin" \
   -e "MINIO_ROOT_PASSWORD=minioadmin" \
   quay.io/minio/minio server /data --console-address ":9090"