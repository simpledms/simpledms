#!/bin/sh
# https://atlasgo.io/guides/migration-tools/golang-migrate

atlas migrate hash --dir "file://db/enttenant/migrate/migrations" --dir-format golang-migrate
atlas migrate hash --dir "file://db/entmain/migrate/migrations" --dir-format golang-migrate
