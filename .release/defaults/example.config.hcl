server {
  data_dir  = "/var/lib/iterator"
  log_level = "DEBUG"
  // Default port is 9095
  listen    = "9095"
  terraform_driver  = "terraform"
  consul {
    address = "localhost:8500"
  }
}

task {
  name        = "Task1"
  description = "Manage infrastructure for web service"
  source      = "path/to/web/terraform/module"
  condition "label-match" {
    // Set to false to prevent non-zero exit codes from this command, from notifying alertmanager that the command failed.
    // Notifying alertmanager (HTTP 500) is likely to re-dispatch the alarm back to am-executor.
    notify_on_failure = false
    // Send a SIGUSR1 signal to the process if it's still running when the triggering alert resolves.
    // Default signal when not specified is SIGKILL.
    resolved_signal   = "SIGUSR1"
    // Don't signal command if a matching 'resolved' message is
    // sent from alertmanager while this command is still running.
    ignore_resolved   = true
    // User defined labels that match an alert
    label {
      alertname = "important_alert"
      severity  = "warning"
      foo       = "bar"
    }
  }
}

task {
  name        = "Task2"
  description = "Manage infrastructure for database service"
  source      = "path/to/database/terraform/module"
  condition "label-match" {
    notify_on_failure = false
    resolved_signal   = "SIGUSR1"
    ignore_resolved   = true
    label {
      alertname = "another_important_alert"
      severity  = "boiling_hot"
      service   = "mysql"
    }
  }
}
