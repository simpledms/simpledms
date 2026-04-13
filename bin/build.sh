#!/bin/sh
CGO_ENABLED=1 go build -tags "sqlite_fts5 sqlite_json sqlite_foreign_keys sqlite_ico" -o simpledms
