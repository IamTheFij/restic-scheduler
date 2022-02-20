job "My App" {
  schedule = "* * * * *"

  config {
    repo = "s3://..."
    passphrase = "foo"
  }

  task "Dump mysql" {
    mysql {
      hostname = "foo"
      username = "bar"
    }
  }

  task "Create biz file" {
    on_backup {
      body = <<EOF
      echo foo > /biz.txt
      EOF
    }
  }

  task "Backup data files" {
    files = [
      "/foo/bar",
      "/biz.txt",
    ]
  }
}
