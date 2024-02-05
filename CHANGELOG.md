## 0.1.3 (February 4, 2024)

IMPROVEMENTS:

* Adding summary state for Iterator status field/key in data directory (or Consul when used as storage backend).
* Adding Sawtooth Terraform scheduling mode for forward-only deployments.
* Adding release CLI subcommand for releasing Terraform resources deployed with the Sawtooth scheduling mode (e.g destroying Terraform resources).
* Adding Terragrunt driver at the global and task configuration level.
* Upgrade to alert digestion concurrency from alert block to individual alerts.
* Restructuring of data directory for alert details storage for basic Consul-Terraform-Sync integration.


BUG FIXES:

* No bug fix for this release.
