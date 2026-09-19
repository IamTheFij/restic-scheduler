// Example job file
job "MyApp" {
  schedule = "* * * * *"

  config {
    repo = "s3://..."
    passphrase = "foo"
    env = {
      "foo" = "bar",
    }
    options {
      verbose_level = 3
    }
  }

  task "Backup main db" {
    mysql "DumpMainDB" {
      hostname = "foo"
      username = "bar"
      dump_to = "/data/main.sql"
    }
  }

  task "Backup Sqlite" {
    sqlite "DumpSqlite" {
      path = "/db/sqlite.db"
      dump_to = "/data/sqlite.db.bak"
    }
  }

  task "Create biz file" {

    pre_script {
      on_backup = <<EOF
      echo bar >> /biz.txt
      EOF
    }

    post_script {
      on_backup = <<EOF
      rm /biz.txt
      EOF
    }
  }

  task "Run restore shell script" {
    pre_script {
      on_restore = "/foo/bar.sh"
    }
  }

  backup {
    files =[
      "/data",
      "/biz.txt",
    ]

    backup_opts {
      tags = ["service"]
    }

    restore_opts {
      verify = true
      # Since paths are absolute, restore to root
      target = "/"
    }
  }

  forget {
    keep_last = 3
    keep_weekly = 2
    keep_monthly = 2
    keep_yearly = 2
    prune = true
  }
}
