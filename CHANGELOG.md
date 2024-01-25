## 1.7.3 (January 15, 2024)

IMPROVEMENTS:

* build: update to go 1.21.6 [[GH-19709](https://github.com/hashicorp/nomad/issues/19709)]
* cgroupslib: Consider CGroups OFF when essential controllers are missing [[GH-19176](https://github.com/hashicorp/nomad/issues/19176)]
* cli: Add new option `nomad setup vault -check` to help cluster operators migrate to workload identities for Vault [[GH-19720](https://github.com/hashicorp/nomad/issues/19720)]
* consul: Add fingerprint for Consul Enterprise admin partitions [[GH-19485](https://github.com/hashicorp/nomad/issues/19485)]
* consul: Added support for Consul Enterprise admin partitions [[GH-19665](https://github.com/hashicorp/nomad/issues/19665)]
* consul: Added support for failures_before_warning and failures_before_critical in Nomad agent services [[GH-19336](https://github.com/hashicorp/nomad/issues/19336)]
* consul: Added support for failures_before_warning in Consul service checks [[GH-19336](https://github.com/hashicorp/nomad/issues/19336)]
* drivers/exec: Added support for OOM detection in exec driver [[GH-19563](https://github.com/hashicorp/nomad/issues/19563)]
* drivers: Enable configuring a raw_exec task to not have an upper memory limit [[GH-19670](https://github.com/hashicorp/nomad/issues/19670)]
* identity: Added vault_role to JWT workload identity claims if specified in jobspec [[GH-19535](https://github.com/hashicorp/nomad/issues/19535)]
* ui: Added group name to allocation tooltips on job status panel [[GH-19601](https://github.com/hashicorp/nomad/issues/19601)]
* ui: Adds a warning message to pages in the Web UI when logs are disabled [[GH-18823](https://github.com/hashicorp/nomad/issues/18823)]
* ui: Hide token secret upon successful login [[GH-19529](https://github.com/hashicorp/nomad/issues/19529)]
* ui: when an Action has long output, anchor to the latest messages [[GH-19452](https://github.com/hashicorp/nomad/issues/19452)]
* vault: Add `allow_token_expiration` field to allow Vault tokens to expire without renewal for short-lived tasks [[GH-19691](https://github.com/hashicorp/nomad/issues/19691)]
* vault: Nomad clients will no longer attempt to renew Vault tokens that cannot be renewed [[GH-19691](https://github.com/hashicorp/nomad/issues/19691)]

BUG FIXES:

* acl: Fixed a bug where 1.5 and 1.6 clients could not access Nomad Variables and Services via templates [[GH-19578](https://github.com/hashicorp/nomad/issues/19578)]
* acl: Fixed auth method hashing which meant changing some fields would be silently ignored [[GH-19677](https://github.com/hashicorp/nomad/issues/19677)]
* auth: Added new optional OIDCDisableUserInfo setting for OIDC auth provider [[GH-19566](https://github.com/hashicorp/nomad/issues/19566)]
* client: Fixed a bug where where the environment variable / file for the Consul token weren't written. [[GH-19490](https://github.com/hashicorp/nomad/issues/19490)]
* consul (Enterprise): Fixed a bug where the group/task Consul cluster was assigned "default" when unset instead of the namespace-governed value
* core: Ensure job HCL submission data is persisted and restored during the FSM snapshot process [[GH-19605](https://github.com/hashicorp/nomad/issues/19605)]
* namespaces: Failed delete calls no longer return success codes [[GH-19483](https://github.com/hashicorp/nomad/issues/19483)]
* rawexec: Fixed a bug where oom_score_adj would be inherited from Nomad client [[GH-19515](https://github.com/hashicorp/nomad/issues/19515)]
* server: Fix panic when validating non-service reschedule block [[GH-19652](https://github.com/hashicorp/nomad/issues/19652)]
* server: Fix server not waiting for workers to submit nacks for dequeued evaluations before shutting down [[GH-19560](https://github.com/hashicorp/nomad/issues/19560)]
* state: Fixed a bug where purged jobs would not get new deployments [[GH-19609](https://github.com/hashicorp/nomad/issues/19609)]
* ui: Fix rendering of allocations table for jobs that don't have actions [[GH-19505](https://github.com/hashicorp/nomad/issues/19505)]
* vault: Fixed a bug that could cause errors during leadership transition when migrating to the new JWT and workload identity authentication workflow [[GH-19689](https://github.com/hashicorp/nomad/issues/19689)]
* vault: Fixed a bug where `allow_unauthenticated` was enforced when a `default_identity` was set [[GH-19585](https://github.com/hashicorp/nomad/issues/19585)]

## 1.7.2 (December 13, 2023)

FEATURES:

* **Reschedule on Lost**: Adds the ability to prevent tasks on down nodes from being rescheduled [[GH-16867](https://github.com/hashicorp/nomad/issues/16867)]

IMPROVEMENTS:

* audit (Enterprise): Added ACL token role links to audit log auth objects [[GH-19415](https://github.com/hashicorp/nomad/issues/19415)]
* ui: Added a new example template with Task Actions [[GH-19153](https://github.com/hashicorp/nomad/issues/19153)]
* ui: dont allow new jobspec download until template is populated, and remove group count from jobs index [[GH-19377](https://github.com/hashicorp/nomad/issues/19377)]
* ui: make the exec window look nicer on mobile screens [[GH-19332](https://github.com/hashicorp/nomad/issues/19332)]

BUG FIXES:

* auth: Fixed a bug where `tls.verify_server_hostname=false` was not respected, leading to authentication failures between Nomad agents [[GH-19425](https://github.com/hashicorp/nomad/issues/19425)]
* cli: Fix a bug in the `var put` command which prevented combining items as CLI arguments and other parameters as flags [[GH-19423](https://github.com/hashicorp/nomad/issues/19423)]
* client: Fix a panic in building CPU topology when inaccurate CPU data is provided [[GH-19383](https://github.com/hashicorp/nomad/issues/19383)]
* client: Fixed a bug where clients are unable to detect CPU topology in certain conditions [[GH-19457](https://github.com/hashicorp/nomad/issues/19457)]
* consul (Enterprise): Fixed a bug where implicit Consul constraints were not specific to non-default Consul clusters [[GH-19449](https://github.com/hashicorp/nomad/issues/19449)]
* consul: uses token namespace to fetch policies for verification [[GH-18516](https://github.com/hashicorp/nomad/issues/18516)]
* core: Fixed a bug where linux nodes with no reservable cores would panic the scheduler [[GH-19458](https://github.com/hashicorp/nomad/issues/19458)]
* csi: Added validation to `csi_plugin` blocks to prevent `stage_publish_base_dir` from being a subdirectory of `mount_dir` [[GH-19441](https://github.com/hashicorp/nomad/issues/19441)]
* metrics: Revert upgrade of `go-metrics` to fix an issue where metrics from dependencies, such as raft, were no longer emitted [[GH-19374](https://github.com/hashicorp/nomad/issues/19374)]
* ui: Fixed an issue where Accessor ID was masked by default when editing a token [[GH-19432](https://github.com/hashicorp/nomad/issues/19432)]
* vault: Fixed a bug that caused `template` blocks to ignore Nomad configuration for Vault and use the default address of `https://127.0.0.1:8200` when the job does not have a `vault` block defined [[GH-19439](https://github.com/hashicorp/nomad/issues/19439)]

## 1.7.1 (December 08, 2023)

BUG FIXES:

* cli: Fixed a bug that caused the `nomad agent` command to ignore the `VAULT_TOKEN` and `VAULT_NAMESPACE` environment variables [[GH-19349](https://github.com/hashicorp/nomad/issues/19349)]
* client: remove incomplete allocation entries from client state database during client restarts [[GH-16638](https://github.com/hashicorp/nomad/issues/16638)]
* connect: Fixed a bug where deployments would not wait for Connect sidecar task health checks to pass [[GH-19334](https://github.com/hashicorp/nomad/issues/19334)]
* keyring: Fixed a bug where RSA keys were not replicated to followers [[GH-19350](https://github.com/hashicorp/nomad/issues/19350)]
