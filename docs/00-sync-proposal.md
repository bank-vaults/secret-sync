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
* Enable secret synchronization between two secret store services (e.g. between Vault and AWS).
* Provide ways to select which keys need to be synced from source store using either static values, dynamic query, or both.
* Provide a way to transform each key before being sent to destination store.
* Allow concurrent synchronization.
* Support simple sync auditing for transparency.
* Expose functionalities as a standalone CLI
* Provide ways to run inside Kubernetes

## High-level overview
The API is composed of two schemas:
  1. `SecretStore` schema provides access to various secret stores.
     It is composed of `Provider` which specifies the secret store backend (e.g. Vault, AWS),
     and `Permissions` to ensure operational scope (e.g. Read, Write, ReadWrite) on the store itself.

  2. `SyncJob` exposes options for periodic (CRON scheduled) secret synchronization between `Source` and `Dest` store.
     The selection and transformations of secrets to sync can be done via `Plan` list, using:
      * `Secret` - to specify a static secret key
      * `Query` - to specify a dynamic query used to list secret keys to sync from Source
      * `KeyTransform` - to specify ways to transform referenced key (either from secret or query)
      * `Source` - to override default source (future implementation for many-to-1, currently we only focus on 1-to-1 store syncs)

## Example usages
* Synchronize secrets from main Vault instance to local k8s Vault instance every hour
