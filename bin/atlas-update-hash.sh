#!/bin/sh
# https://atlasgo.io/guides/migration-tools/golang-migrate

atlas migrate hash --dir "file://enttenant/migrate/migrations" --dir-format golang-migrate
atlas migrate hash --dir "file://entmain/migrate/migrations" --dir-format golang-migrate
