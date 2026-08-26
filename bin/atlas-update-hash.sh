#!/bin/sh
# https://atlasgo.io/guides/migration-tools/golang-migrate

required_version=$(go list -m -f '{{.Version}}' ariga.io/atlas) || exit 1
installed_version=$(atlas version 2>/dev/null) || {
	echo "atlas is not installed" >&2
	exit 1
}
case "$installed_version" in
	"atlas version $required_version"*) ;;
	*)
		echo "atlas $required_version is required; installed version output:" >&2
		echo "$installed_version" >&2
		exit 1
		;;
esac

atlas migrate hash --dir "file://db/enttenant/migrate/migrations" --dir-format golang-migrate
atlas migrate hash --dir "file://db/entmain/migrate/migrations" --dir-format golang-migrate
