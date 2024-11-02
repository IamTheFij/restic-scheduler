# [restic-scheduler](/iamthefij/restic-scheduler)

## About

`restic-scheduler` is a tool designed to allow declarative scheduling of restic backups using HCL (HashiCorp Configuration Language). This tool simplifies the process of managing and automating backups by defining jobs in a configuration file.

## Getting Started

### Installation

You can install `restic-scheduler` using the following command:

```sh
go install git.iamthefij.com/iamthefij/restic-scheduler@latest
```

You can also download the latest release from the [releases page](https://git.iamthefij.com/iamthefij/restic-scheduler/releases).

Finally, if you prefer to use Docker, you can run something like the following command:

```sh
docker run -v /path/to/config:/config -v /path/to/data:/data iamthefij/restic-scheduler -config /config/jobs.hcl
```

### Prerequisites

If you're not using Docker, you'll need to ensure that `restic` is installed and available in your system's PATH. You can download and install restic from [here](https://restic.net/).

## Usage

### Command Line Interface

The `restic-scheduler` command line interface provides several options for managing backup, restore, and unlock jobs. Below are some examples of how to use this tool.

#### Display Version

To display the version of `restic-scheduler`, use the `-version` flag:

```sh
restic-scheduler -version
```

#### Run Backup Jobs

To run backup jobs, use the `-backup` flag followed by a comma-separated list of job names. Use `all` to run all backup jobs:

```sh
restic-scheduler -backup job1,job2
```

#### Run Restore Jobs

To run restore jobs, use the `-restore` flag followed by a comma-separated list of job names. Use `all` to run all restore jobs:

```sh
restic-scheduler -restore job1,job2
```

#### Unlock Job Repositories

To unlock job repositories, use the `-unlock` flag followed by a comma-separated list of job names. Use `all` to unlock all job repositories:

```sh
restic-scheduler -unlock job1,job2
```

#### Run Jobs Once and Exit

To run specified backup and restore jobs once and exit, use the `-once` flag:

```sh
restic-scheduler -backup job1 -restore job2 -once
```

#### Health Check and metrics API

To bind the health check and Prometheus metrics API to a specific address, use the `-addr` flag:

```sh
restic-scheduler -addr 0.0.0.0:8080
```

#### Metrics Push Gateway

To specify the URL of a Prometheus push gateway service for batch runs, use the `-push-gateway` flag:

```sh
restic-scheduler -push-gateway http://example.com
```

## HCL Configuration

The configuration for `restic-scheduler` is defined using HCL. Below is a description and example of how to define a backup job in the configuration file.

### Job Configuration

A job in the configuration file is defined using the `job` block. Each job must have a unique name, a schedule, and a configuration for restic. Additionally, tasks can be defined to perform specific actions before and after the backup.

#### Fields

- `name`: The name of the job.
- `schedule`: The cron schedule for the job.
- `config`: The restic configuration block.
  - `repo`: The restic repository.
  - `passphrase`: (Optional) The passphrase for the repository.
  - `env`: (Optional) Environment variables for restic.
  - `options`: (Optional) Global options for restic. See the `restic` command for details.
- `task`: (Optional) A list of tasks to run before and after the backup.
- `mysql`, `postgres`, `sqlite`: (Optional) Database-specific tasks.
- `backup`: The backup configuration block.
- `forget`: (Optional) Options for forgetting old snapshots.

### Example

Below is an example of a job configuration in HCL:

```hcl
// Example job file
job "MyApp" {
  schedule = "* * * * *"

  config {
    repo = "s3://..."
    passphrase = "foo"
    # Some alternate ways to pass the passphrase to restic
    # passphrase = env("RESTIC_PASSWORD")
    # passphrase = readfile("/path/to/passphrase")
    env = {
      "foo" = "bar",
    }
    options {
      VerboseLevel = 3
      # Another alternate way to pass the passphrase to restic
      # PasswordFile = "/path/to/passphrase"
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
```

```sh
restic-scheduler jobs.hcl
```

This will read the job definitions from `jobs.hcl` and execute the specified jobs.

For more examples, check out `./config.hcl` or some of the example integration test configs in `./test/`.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on the [GitHub repository](https://git.iamthefij.com/iamthefij/restic-scheduler).

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
