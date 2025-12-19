env "main" {
  migration {
    dir = "file://entmain/migrate/migrations"
    format = "golang-migrate"
  }
}

env "tenant" {
  migration {
    dir = "file://enttenant/migrate/migrations"
    format = "golang-migrate"
  }
}
