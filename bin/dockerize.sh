#!/bin/sh
docker build --ssh default -f Dockerfile -t ghcr.io/simpledms/simpledms .
docker tag ghcr.io/simpledms/simpledms:latest simpledms/simpledms:latest