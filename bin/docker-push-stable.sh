#!/bin/sh
docker tag ghcr.io/marcobeierer/simpledms:latest ghcr.io/marcobeierer/simpledms:stable
docker push ghcr.io/marcobeierer/simpledms:stable
