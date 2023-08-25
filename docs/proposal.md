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
* Provide ways to select which keys need to be synced from source store using:
  * a list of static keys (will not be synced if they do not exist on source when performing sync)
  * a list of regex filters (will be applied to source store List API response to create a dynamic list of keys before performing sync)
* It is possible to combine keys and list filters to cover a common scenario such as "sync secrets with these keys, and any additional
  secrets whose key matches one of these filters".
* Provide a way to transform per-key update request before being sent to destination store.
* Allow concurrent synchronization.
* Support simple sync auditing for transparency.
* Expose functionalities as a standalone CLI, as well as a Kubernetes Operator.

## High-level overview
The API is composed of two schemas:
  1. `SecretStore` schema provides access to various secret stores.
     It is composed of `Provider` which specifies the secret store backend (e.g. Vault, AWS),
     and `Permissions` to ensure operational scope (e.g. Read, Write, ReadWrite) on the store itself.

  2. `SyncJob` exposes `Once` and `Schedule` options for single and periodic (CRON scheduled) secret synchronization between `Source` and `Dest` store.
     The selection of secrets to sync can be done by specifying a static key list using `Keys`, or a list of regex filters via `ListFilters`.
     Finally, transformations of update request key-objects is supported via `Template`.

## Proposal
### SecretStore
```golang
type SecretStorePermissions string

type SecretStoreProvider struct {
    // Only one provider can be set
    ProviderA *SecretStoreProviderA `json:"provider-a,omitempty"`
	
    // ProviderB *SecretStoreProviderB `json:"provider-b,omitempty"`

}

type SecretStore struct {
    // Used to configure store mode. Defaults to ReadWrite.
    // Optional
    Permissions SecretStorePermissions `json:"permissions,omitempty"`
    
    // Used to configure secrets provider.
    // Required
    Provider SecretStoreProvider `json:"provider"`
}
```

### SyncJob
```golang
type SyncJob struct {
    // Used to configure the source for sync request.
    // Required
    SourceStore SecretStore `json:"source-store"`
    
    // Used to configure the destination for sync request.
    // Required
    DestStore SecretStore `json:"dest-store"`
    
    // Used to explicitly specify which keys to sync.
    // Optional
    Keys []StoreKey `json:"keys,omitempty"`
    
    // Defines regex filters used to dynamically get additional keys to sync.
    // Optional
    ListFilters []string `json:"list-filters,omitempty"`
    
    // Template is applied to every key struct before sync.
    // Must return a valid JSON StoreKey object.
    // Optional
    Template string `json:"template,omitempty"`
    
    // Used to configure schedule for synchronization.
    // Optional
    Schedule string `json:"schedule,omitempty"`
    
    // Used to only perform sync once.
    // If specified, Schedule will be ignored.
    // Optional
    RunOnce *bool `json:"run-once,omitempty"`
    
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
keys:
  - key: a
  - key: b/b
    version: "1"
  - key: c/c/c
    version: "2"
list-filters:
  - "d/d/d/.*"
template: |
  {
    "key": "{{.Key}}/new-key", 
    "version": "2"
  }
schedule: "@every 1h"
```

```bash
secret-sync --source path/to/vault-source.yaml \
            --dest path/to/vault-dest.yaml \
            --sync path/to/sync-job.yaml
```
