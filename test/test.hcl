job "TestBackup" {
  schedule = "* * * * *"

  config {
    repo = "test/data/backups"
    passphrase = "supersecret"

    options {
      CacheDir = "test/data/cache"
    }
  }

  task "create test data" {
    pre_script {
      on_backup = "echo test > test/data/data/test.txt"
    }
  }

  task "backup phases" {
    pre_script {
      on_backup = "echo 'pre-backup'"
      on_restore = "echo 'pre-restore'"
    }

    post_script {
      on_backup = "echo 'post-backup'"
      on_restore = "echo 'post-restore'"
    }
  }

  backup {
    paths = ["./test/data/data"]
    restore_opts {
      Target = "."
    }
  }

  forget {
    KeepLast = 2
  }
}
