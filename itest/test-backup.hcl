job "IntegrationTest" {
  schedule = "@daily"

  config {
    repo = "/repo"
    passphrase = "shh"

    env = {
      # Env to validate is usable in tasks
      hello_prebackup = "Hello"
      hello_prerestore = "HelloPreRestore"
    }
  }

  task "Basic script task" {
    pre_script {
      env = {
        # To verify that this value takes precedence over the config value
        hello_prebackup = "HelloPreBackup"
      }

      on_backup = <<EOF
      echo "$hello_prebackup" > /data/pre-backup.txt
      EOF

      on_restore = <<EOF
      echo "$hello_prerestore" > /data/pre-restore.txt
      echo "Pre" > /data/on-restore.txt
      EOF
    }

    post_script {
      on_backup = <<EOF
      echo "Hello" > /data/post-backup.txt
      EOF
      on_restore = <<EOF
      echo "Post" >> /data/on-restore.txt
      EOF
    }
  }

  mysql "MySQL" {
    hostname = env("MYSQL_HOST")
    database = "main"
    username = env("MYSQL_USER")
    password = env("MYSQL_PWD")
    dump_to = "/tmp/mysql.sql"
    use_mariadb = true
  }

  postgres "Postgres" {
    hostname = env("PGSQL_HOST")
    database = "main"
    username = env("PGSQL_USER")
    password = env("PGSQL_PASS")
    create = true
    dump_to = "/tmp/psql.sql"
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
