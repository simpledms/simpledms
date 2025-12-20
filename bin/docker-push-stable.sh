#!/bin/sh
docker tag ghcr.io/simpledms/simpledms:latest ghcr.io/simpledms/simpledms:stable
docker push ghcr.io/simpledms/simpledms:stable
