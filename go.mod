module github.com/simpledms/simpledms

go 1.25

// TODO add ent and maybe atlas
tool (
	// doesn't work at the moment, see
	// https://github.com/ent/ent/issues/4464
	// entgo.io/ent/cmd/ent
	github.com/air-verse/air
	github.com/marcobeierer/enumer
	golang.org/x/text/cmd/gotext
)

require (
	ariga.io/atlas v1.1.0
	entgo.io/ent v0.14.5
	filippo.io/age v1.3.1
	github.com/Shopify/toxiproxy v2.1.4+incompatible
	github.com/cyphar/filepath-securejoin v0.6.1
	github.com/go-playground/form v3.1.4+incompatible
	github.com/go-playground/validator/v10 v10.30.1
	github.com/go-task/slim-sprig/v3 v3.0.0
	github.com/golang-migrate/migrate/v4 v4.19.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/handlers v1.5.2
	github.com/iancoleman/strcase v0.3.0
	github.com/joho/godotenv v1.5.1
	github.com/marcobeierer/go-tika v1.0.1
	github.com/marcobeierer/structs v1.0.0
	github.com/matoous/go-nanoid v1.5.1
	github.com/mattn/go-sqlite3 v1.14.33
	github.com/minio/minio-go/v7 v7.0.98
	github.com/pquerna/otp v1.5.0
	github.com/puzpuzpuz/xsync/v4 v4.3.0
	github.com/wneessen/go-mail v0.7.2
	github.com/yuin/goldmark v1.7.16
	golang.org/x/crypto v0.47.0
	golang.org/x/term v0.39.0
	golang.org/x/text v0.33.0
)

require (
	dario.cat/mergo v1.0.2 // indirect
	filippo.io/hpke v0.4.0 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/air-verse/air v1.63.4 // indirect
	github.com/apparentlymart/go-textseg/v15 v15.0.0 // indirect
	github.com/bep/godartsass/v2 v2.5.0 // indirect
	github.com/bep/golibsass v1.2.0 // indirect
	github.com/bmatcuk/doublestar v1.3.4 // indirect
	github.com/boombuler/barcode v1.0.2 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/fatih/color v1.18.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.12 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-openapi/inflect v0.21.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gohugoio/hugo v0.149.1 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/hashicorp/hcl/v2 v2.23.0 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.11 // indirect
	github.com/klauspost/crc32 v1.3.0 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/marcobeierer/enumer v0.0.0-20250424083623-3196a84fb274 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/minio/crc64nvme v1.1.1 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/mitchellh/go-wordwrap v1.0.1 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/philhofer/fwd v1.2.0 // indirect
	github.com/rs/xid v1.6.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/tdewolff/parse/v2 v2.8.3 // indirect
	github.com/tinylib/msgp v1.6.1 // indirect
	github.com/zclconf/go-cty v1.16.2 // indirect
	github.com/zclconf/go-cty-yaml v1.1.0 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/mod v0.31.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/tools v0.40.0 // indirect
	google.golang.org/protobuf v1.36.8 // indirect
	gopkg.in/go-playground/assert.v1 v1.2.1 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
