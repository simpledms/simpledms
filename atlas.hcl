env "main" {
  migration {
    dir = "file://db/entmain/migrate/migrations"
    format = "golang-migrate"
  }
}

env "tenant" {
  migration {
    dir = "file://db/enttenant/migrate/migrations"
    format = "golang-migrate"
  }
}
