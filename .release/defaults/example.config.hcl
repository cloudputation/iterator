server {
  data_dir  = "/var/lib/iterator"
  log_dir   = "/var/log/iterator"
  log_level = "info"
  // Default port is 9595
  listen    = "9595"
  terraform_driver  = "terraform"
  consul {
    address = "10.100.200.210:8500"
  }
}

task {
  name        = "Task1"
  description = "Manage infrastructure for web service"
  source      = "/var/lib/iterator/terraform-data/moduleA"
  condition "label-match" {
    // Set to false to prevent non-zero exit codes from this command, from notifying alertmanager that the command failed.
    // Notifying alertmanager (HTTP 500) is likely to re-dispatch the alarm back to am-executor.
    notify_on_failure = false

    // Send a SIGUSR1 signal to the process if it's still running when the triggering alert resolves.
    // Default signal when not specified is SIGKILL.
    resolved_signal   = "SIGKILL"

    // Don't signal command if a matching 'resolved' message is
    // sent from alertmanager while this command is still running.
    ignore_resolved   = true

    // User defined labels that match an alert
    label {
      alertname = "my_cool_alert"
      severity  = "warning"
    }
  }
}

task {
  name        = "Task2"
  description = "Manage infrastructure for database service"
  source      = "/var/lib/iterator/terraform-data/moduleB"
  condition "label-match" {
    notify_on_failure = false
    resolved_signal   = "SIGUSR1"
    ignore_resolved   = true
    label {
      alertname = "the_other_cool_alert"
      severity  = "boiling_hot"
    }
  }
}
