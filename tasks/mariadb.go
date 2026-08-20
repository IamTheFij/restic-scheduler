package tasks

// JobTaskMariaDB is a MySQL backup task that performs required pre and post tasks.
type JobTaskMariaDB struct {
	JobTaskMySQL `hcl:",remain"`

	// labels won't pass through with remain
	Name string `hcl:"name,label"`
}

func (t *JobTaskMariaDB) patchInnerStruct() {
	// Hook in GetPreTask to set to mariaDB
	t.UseMariaDB = true
	// Copy name down to inner struct
	t.JobTaskMySQL.Name = t.Name
}

func (t JobTaskMariaDB) GetPreTask() ExecutableTask {
	t.patchInnerStruct()

	return t.JobTaskMySQL.GetPreTask()
}

func (t JobTaskMariaDB) GetPostTask() ExecutableTask {
	t.patchInnerStruct()

	return t.JobTaskMySQL.GetPostTask()
}

func (t JobTaskMariaDB) Validate() error {
	t.patchInnerStruct()

	return t.JobTaskMySQL.Validate()
}
