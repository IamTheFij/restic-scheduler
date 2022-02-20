package main

// JobConfig is all configuration to be sent to Restic
type JobConfig struct {
	Repo       string            `hcl:"repo"`
	Passphrase string            `hcl:"passphrase"`
	Env        map[string]string `hcl:"env,optional"`
	Args       []string          `hcl:"args"`
}

// JobTaskScript is a sript to be executed as part of a job task
type JobTaskScript struct {
	ScriptPath string `hcl:"path,optional"`
	Body       string `hcl:"body,optional"`
}

// JobTaskMySQL is a sqlite backup task that performs required pre and post tasks
type JobTaskMySQL struct {
	Hostname string `hcl:"hostname,optional"`
	Database string `hcl:"database,optional"`
	Username string `hcl:"username,optional"`
	Password string `hcl:"password,optional"`
}

// JobTaskSqlite is a sqlite backup task that performs required pre and post tasks
type JobTaskSqlite struct {
	Path string `hcl:"path,label"`
}

// JobTask represents a single task within a backup job
type JobTask struct {
	Name      string          `hcl:"name,label"`
	OnBackup  []JobTaskScript `hcl:"on_backup,block"`
	OnRestore []JobTaskScript `hcl:"on_restore,block"`
	MySql     []JobTaskMySQL  `hcl:"mysql,block"`
	Sqlite    []JobTaskSqlite `hcl:"sqlite,block"`
	Files     []string        `hcl:"files,optional"`
}

// Job contains all configuration required to construct and run a backup
// and restore job
type Job struct {
	Name     string    `hcl:"name,label"`
	Schedule string    `hcl:"schedule"`
	Config   JobConfig `hcl:"config,block"`
	Tasks    []JobTask `hcl:"task,block"`
	Validate bool      `hcl:"validate,optional"`
}

func (job Job) NewRestic() ResticCmd {
	return ResticCmd{
		LogPrefix:  job.Name,
		Repo:       job.Config.Repo,
		Env:        job.Config.Env,
		Passphrase: job.Config.Passphrase,
	}
}

type Config struct {
	Jobs []Job `hcl:"job,block"`
}

/***

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

***/
