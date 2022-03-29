// A simple backup job
job "BackupDataDir" {
  schedule = "@daily"

  config {
    repo = "./backups"
    passphrase = "secret phrase"
  }

  backup {
    paths = ["./data"]

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      Target = "."
    }

  }

  forget {
    KeepLast = 2
    Prune = true
  }
}

job "PassphraseFile" {
  schedule = "@daily"

  config {
    repo = "./backups"
    options {
      // A more secure method of specifying password
      PasswordFile = "./test/samplepassphrase.txt"
    }
  }

  backup {
    paths = ["./data"]

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      Target = "."
    }

  }
}

job "BackupDataAndSqlite" {
  schedule = "@daily"

  config {
    repo = "./backups"
    // Another safe way of not inlining the passphrase
    passphrase = readfile("./test/samplepassphrase.txt")
  }

  sqlite "Backup database" {
    path = "./sqlite.db"
    dump_to = "./data/sqlite.db.bak"
  }

  backup {
    paths = ["./data"]

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      Target = "."
    }
  }
}

job "BackupMySQLDatabase" {
  schedule = "@daily"

  config {
    repo = "./backups"
    passphrase = "secret phrase"
  }

  mysql "Backup database" {
    hostname = "localhost"
    database = "dbname"
    username = "username"
    // Values can be read from the env to avoid inlining as well
    password = env("TEST_PASSWORD")
    dump_to = "./data/sqlite.db.bak"
  }

  backup {
    paths = ["./data"]

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      Target = "."
    }
  }
}
