## Secret Sync

Enables secret synchronization between two secret store services (e.g. between Vault and AWS) in a configurable manner.

> [!WARNING]  
> This is an early alpha version and there will be changes made to the API. You can support us with your feedback.

### Supported secret stores
- Vault
- FileDir (regular system directory)

### Quick usage
Synchronize secrets every hour from Vault-A to Vault-B instance.

#### Define stores and sync job strategy
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
  - secret:
      key: b/b
  - secret:
      key: c/c/cÏ
  - query:
      path: "d/d/d"
      key:
        regexp: .*
    key-transform:
      - regexp:
          source: "d/d/d/(.*)"
          target: "d/d/d/$1-final"
```

#### Perform sync
```bash
secret-sync --source path/to/vault-source.yaml \
            --dest path/to/vault-dest.yaml \
            --sync path/to/sync-job.yaml
# Use --schedule "@every 1m" to override sync job file config.
```

### Docs
Check documentation and example usage at [PROPOSAL](docs/proposal.md).
