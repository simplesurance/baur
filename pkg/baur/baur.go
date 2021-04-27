// Package baur implements an incremental task runner.
//
// A directory and child-directories containing baur configuration files is
// called a repository. The root of the repository contains the RepositoryCfgFile.
// The root and each child directory can contain an application with an AppCfgFile.
//
// Each AppCfgFile can define one or more tasks. A task consist of a Command to
// run, Input definitions and optionally Output definitions.
// An Input is a file or string that when it changes, it changes the result of
// the executed command. An output is a something that is created when the task
// is executed. Outputs define remote upload locations to which they should be
// uploaded too.
// The results of task runs together with the state of the inputs are recorded
// in a storage. The state of inputs is tracked by calculating a digest for all
// inputs of a task.
// With the records about past task runs in the storage and the digest of the
// inputs of a task, baur evaluates if a task has been run with the same inputs
// in the past already. If it has not, it's executions is pending.
//
// Basic Workflow
//
// - Loader: Locate and load a repository configuration file, discover
// applications and load and parse their configuration files.
//
// - TaskStatusEvaluator: Query the storage and determine which applications
// have not been run before with their current inputs.
//
// - TaskRunner: Execute the commands for the tasks that should be run.
//
// - Uploader: Uploads the outputs that the tasks produced.
//
// - StoreRun: Record the task executions the uploaded outputs in the database.
package baur

// AppCfgFile is the name of application configuration files.
const AppCfgFile = ".app.toml"

// RepositoryCfgFile is the name of the repository configuration file.
const RepositoryCfgFile = ".baur.toml"
