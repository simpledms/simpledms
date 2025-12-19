#!/bin/sh
# CGO_ENABLED=1 go install --tags fts5 github.com/go-jet/jet/v2/cmd/jet@latest
jet -source=sqlite -dsn=".testdata/.simpledms/simpledms.sqlite3" -schema=public -path=./jet

