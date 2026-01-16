#!/bin/sh
docker tag ghcr.io/simpledms/simpledms:latest ghcr.io/simpledms/simpledms:stable
docker push ghcr.io/simpledms/simpledms:stable

docker tag simpledms/simpledms:latest simpledms/simpledms:stable
docker push simpledms/simpledms:stable

echo "don't forget to create a release tag"