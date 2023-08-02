job "IntegrationTest" {
  schedule = "@daily"

  config {
    repo = "/repo"
    passphrase = "shh"
  }

  mysql "MySQL" {
    hostname = env("MYSQL_HOST")
    database = "main"
    username = env("MYSQL_USER")
    password = env("MYSQL_PWD")
    dump_to = "/tmp/mysql.sql"
  }

  sqlite "SQLite" {
    path = "/data/test_database.db"
    dump_to = "/data/test_database.db.bak"
  }

  backup {
    paths = ["/data"]

    restore_opts {
      Target = "/"
    }
  }
}
