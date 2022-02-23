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
  }

  sqlite "DumpSqlite" {
    path = "/db/path"
  }

  task "RunSomePreScripts" {
    script {
      on_backup = <<EOF
      echo foo > /biz.txt
      EOF

      on_restore = "/foo/bar.sh"
    }

    script {
      on_backup = <<EOF
      echo bar >> /biz.txt
      EOF
    }
  }

  task "ActuallyBackupSomeStuff" {
    backup {
      files =[
        "/foo/bar",
        "/biz.txt",
      ]

      backup_opts {
        Tags = ["service"]
      }

      restore_opts {
        Verify = true
      }
    }
  }

  task "RunSomePostScripts" {
    script {
      on_backup = <<EOF
      rm /biz.txt
      EOF
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
