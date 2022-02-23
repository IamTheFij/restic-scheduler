job "TestBackup" {
  schedule = "1 * * * *"

  config {
    repo = "./backups"
    passphrase = "supersecret"

    options {
      CacheDir = "./cache"
    }
  }

  task "before script" {
    script {
      on_backup = "echo before backup!"
    }
  }

  task "backup" {
    backup {
      files = [
        "./data"
      ]

      backup_opts {
        Tags = ["foo"]
      }
    }
  }

  task "after script" {
    script {
      on_backup = "echo after backup!"
    }
  }

  forget {
    KeepLast = 2
  }
}
