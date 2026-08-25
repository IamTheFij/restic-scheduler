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
      target = "."
    }

  }

  forget {
    keep_last = 2
    prune = true
  }
}

job "PassphraseFile" {
  schedule = "@daily"

  config {
    repo = "./backups"
    options {
      // A more secure method of specifying password
      password_file = "./test/samplepassphrase.txt"
    }
  }

  backup {
    paths = ["./data"]

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      target = "."
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

  task "backup database" {
    sqlite "Backup database" {
      path = "./sqlite.db"
      dump_to = "./data/sqlite.db.bak"
    }
  }

  backup {
    paths = ["./data"]

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      target = "."
    }
  }
}

job "BackupMySQL" {
  schedule = "@daily"

  config {
    repo = "./backups"
    passphrase = "secret phrase"
  }

  task "Backup database" {
    mysql "Backup database" {
      hostname = "localhost"
      database = "dbname"
      username = "username"
      // Values can be read from the env to avoid inlining as well
      password = env("TEST_PASSWORD")
      dump_to = "dump.sql"
    }
  }

  backup {
    // Test empty path list since path should be added by database task
    paths = []

    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      target = "."
    }
  }
}

job "BackupMariaDB" {
  schedule = "@daily"

  config {
    repo = "./backups"
    passphrase = "secret phrase"
  }

  task "Backup database" {
    mariadb "Backup database" {
      hostname = "localhost"
      database = "dbname"
      username = "username"
      // Values can be read from the env to avoid inlining as well
      password = env("TEST_PASSWORD")
      dump_to = "dump.sql"
    }
  }

  backup {
    // Test missing paths since path should be added by database task
    paths = []
    restore_opts {
      // Since backup paths are relative to cwd, we're going to restore relative to cwd as well
      target = "."
    }
  }
}
