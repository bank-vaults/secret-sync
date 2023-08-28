Secret Sync
===================

This document describes the operational and support aspect of secret synchronization.

<!-- TOC -->
* [Secret Sync](#secret-sync)
  * [Goals](#goals)
  * [High-level overview](#high-level-overview)
  * [Proposal](#proposal)
    * [SecretStore](#secretstore)
    * [SyncJob](#syncjob)
  * [Potential issues](#potential-issues)
  * [Example usages](#example-usages)
    * [Synchronize secrets from Vault-Source to Vault-Dest instance every hour](#synchronize-secrets-from-vault-source-to-vault-dest-instance-every-hour)
<!-- TOC -->

## Goals
* Enable secret synchronization between two secret store services (e.g. between Vault and AWS) in a configurable manner.
* Provide ways to select which keys need to be synced from source store using either static values, dynamic query, or both.
* Provide a way to transform each key before being sent to destination store.
* Allow concurrent synchronization.
* Support simple sync auditing for transparency.
* Expose functionalities as a standalone CLI, as well as a Kubernetes Operator.

## High-level overview
The API is composed of two schemas:
  1. `SecretStore` schema provides access to various secret stores.
     It is composed of `Provider` which specifies the secret store backend (e.g. Vault, AWS),
     and `Permissions` to ensure operational scope (e.g. Read, Write, ReadWrite) on the store itself.

     2. `SyncJob` exposes `Once` and `Schedule` options for single and periodic (CRON scheduled) secret synchronization between `Source` and `Dest` store.
        The selection and transformations of secrets to sync can be done via `Plan` list, using:
         * `Secret` - to specify a static secret key
         * `Query` - to specify a dynamic query used to list secret keys to sync from Source
         * `[]Rewrite` - to specify regexp group rewrites to apply on secret key (this will be applied to fetched secret keys)
         * `Source` - to override default source (future implementation for many-to-1, currently we only focus on 1-to-1 store syncs)

## Proposal
### SecretStore
```golang
type SecretStore struct {
    // Used to configure store mode. Defaults to ReadWrite.
    // Optional
    Permissions SecretStorePermissions `json:"permissions,omitempty"`
    
    // Used to configure secrets provider.
    // Required
    Provider SecretStoreProvider `json:"provider"`
}

// Only one provider can be specified
type SecretStoreProvider struct {
    ProviderA *SecretStoreProviderA `json:"provider-a,omitempty"`
    
    ProviderB *SecretStoreProviderB `json:"provider-b,omitempty"`
}
```

### SyncJob
```golang
type SyncJobSpec struct {
    // Used to configure the source for sync request.
    // Required
    SourceRef SecretStoreRef `json:"source"`
    
    // Used to configure the destination for sync request.
    // Required
    DestRef SecretStoreRef `json:"dest"`
    
    // Used to configure schedule for synchronization.
    // The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
    // Defaults to @hourly
    // Optional
    Schedule string `json:"schedule,omitempty"`
    
    // Used to only perform sync once.
    // If specified, Schedule will be ignored.
    // Optional
    RunOnce bool `json:"run-once,omitempty"`
    
    // Used to specify sync plan.
    // Required
    Plan []SecretKeyFromRef `json:"plan,omitempty"`
    
    // The number of sync results to retain.
    // Optional
    HistoryLimit *int32 `json:"history-limit,omitempty"`
    
    // Points to a file where all sync logs should be saved to.
    // Optional
    AuditLogPath string `json:"audit-log-path,omitempty"`
}
```

## Potential issues
* Support when dealing with cross-API support for both CLI and K8s which can include conversion and validation
* Complications when overriding default source (if supporting many sources to one destination scenario)
* Handling SyncJobs schedules will be a bit tricky on restarts/crashes

## Example usages
### Synchronize secrets from Vault-Source to Vault-Dest instance every hour
```yaml
### SecretStore: path/to/vault-source.yaml
permissions: Read
provider:
  vault:
    address: http://0.0.0.0:8200
    unseal-keys-path: secret
    role: ''
    auth-path: userpass
    token-path: ''
    token: root
```
```yaml
### SecretStore: path/to/vault-dest.yaml
permissions: Write
provider:
  vault:
    address: http://0.0.0.0:8201
    unseal-keys-path: secret
    role: ''
    auth-path: userpass
    token-path: ''
    token: root
```
```yaml
### SyncJob: path/to/sync-job.yaml
schedule: "@every 1h"
plan:
  - secret:
      key: a
    rewrite:
      - regexp:
          source: "a"
          target: "a-transient"
      - regexp:
          source: "a-transient"
          target: "a-final"
  - secret:
      key: b/b
      version: "1"
  - secret:
      key: c/c/c
      version: "2"
  - query:
      path: "d/d/d"
      key:
        regexp: .*
    rewrite:
      - regexp:
          source: "d/d/d/1"
          target: "d/d/d/1-final"
```

```bash
secret-sync --source path/to/vault-source.yaml \
            --dest path/to/vault-dest.yaml \
            --sync path/to/sync-job.yaml
```
