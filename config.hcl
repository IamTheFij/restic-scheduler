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
      VerboseLevel = 3
    }
  }

  mysql "DumpMainDB" {
    hostname = "foo"
    username = "bar"
    dump_to = "/data/main.sql"
  }

  sqlite "DumpSqlite" {
    path = "/db/sqlite.db"
    dump_to = "/data/sqlite.db.bak"
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
      Tags = ["service"]
    }

    restore_opts {
      Verify = true
      # Since paths are absolute, restore to root
      Target = "/"
    }
  }

  forget {
    KeepLast = 3
    KeepWeekly = 2
    KeepMonthly = 2
    KeepYearly = 2
    Prune = true
  }
}
