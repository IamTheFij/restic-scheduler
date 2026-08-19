job "IntegrationTest" {
  schedule = "* * * * *"

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

  task "Script task adds file to backup" {
    pre_script {
      # Create a file outside our backup task path list
      on_backup = "echo 'howdy' > /tmp/hello.txt"

      # Add the path to the backup paths
      backup_paths = ["/tmp/hello.txt"]
    }

    post_script {
      # This should ensure that we fail if this doesn't exist after restoring
      on_restore = "cat /tmp/hello.txt"
    }
  }

  task "Backup databases" {
    mariadb "MariaDB" {
      hostname = env("MYSQL_HOST")
      username = env("MYSQL_USER")
      password = env("MYSQL_PWD")
      database = "main"
      dump_to = "/tmp/mysql.sql"
      skip_ssl = true
    }

    postgres "Postgres" {
      hostname = env("PGSQL_HOST")
      username = env("PGSQL_USER")
      password = env("PGSQL_PASS")
      database = "main"
      create = true
      dump_to = "/tmp/psql.sql"
    }

    sqlite "SQLite" {
      path = "/data/test_database.db"
      dump_to = "/tmp/test_database.db.bak"
    }
  }

  backup {
    paths = ["/data"]

    restore_opts {
      Target = "/"
    }
  }
}
