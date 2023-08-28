## Secret Sync

Enables secret synchronization between two secret store services (e.g. between Vault and AWS) in a configurable manner.

### Supported secret stores
- Vault

### Quick usage
Synchronize secrets every hour from Vault-A to Vault-B instance.
```yaml
### Vault-A - Source
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
### Vault-B - Dest
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
# Use --schedule "@every 1m" to override sync job file config.
```

### Docs
Check documentation and example usage at [PROPOSAL](docs/proposal.md#Example_usage).
