server {
  data_dir  = "it-data"
  log_level = "DEBUG"
  listen    = ":9095"
  terraform_driver  = "terraform"
  consul {
    address = "localhost:8500"
  }
}

task {
  name        = "Task1"
  description = "Manage infrastructure for web service"
  source      = "tests/env/terraform/moduleA"
  condition "label-match" {
    notify_on_failure = false
    resolved_signal   = "SIGUSR1"
    ignore_resolved   = true
    label {
      alertname = "my_cool_alert"
      severity  = "warning"
    }
  }
}

task {
  name        = "Task2"
  description = "Manage infrastructure for database service"
  source      = "tests/env/terraform/moduleB"
  condition "label-match" {
    notify_on_failure = false
    resolved_signal   = "SIGUSR1"
    ignore_resolved   = true
    label {
      alertname = "the_other_cool_alert"
      severity  = "boiling_hot"
      thistagisimportant = "soisitsvalue"
    }
  }
}
