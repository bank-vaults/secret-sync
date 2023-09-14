## Secret Sync

Enables secret synchronization between two secret store services (e.g. between Hashicorp Vault and AWS) in a configurable and explicit manner.

> [!WARNING]  
> This is an early alpha version and there will be changes made to the API. You can support us with your feedback.

### Supported secret stores
- Hashicorp Vault
- FileDir (store is a folder, secrets are plain unencrypted files)

### Examples

<details>
<summary>Synchronize specific secrets every hour between two Hashicorp Vault instance</summary>

#### Define stores
```yaml
### Vault-A - Source
### SecretStore: path/to/vault-source.yaml
vault:
    address: "http://0.0.0.0:8200"
    storePath: "secret"
    role: ""
    authPath: "userpass"
    tokenPath: ""
    token: "root"
```
```yaml
### Vault-B - Target
### SecretStore: path/to/vault-target.yaml
vault:
    address: "http://0.0.0.0:8201"
    storePath: "secret"
    role: ""
    authPath: "userpass"
    tokenPath: ""
    token: "root"
```

#### Define sync strategy
```yaml
### SyncJob: path/to/sync-job.yaml
schedule: "@every 1h"
## Defines how the secrets will be synced
sync:
  ## 1. Usage: Sync key from ref
  - secretRef:
      key: /source/credentials/username
    target: # If not specified, will be synced under the same key
      key: /target/example-1

  ## 2. Usage: Sync all keys from query
  - secretQuery:
      path: /source/credentials
      key:
        regexp: .*
    target: # If not specified, all keys will be synced under the same path
      keyPrefix: /target/example-2/

  ## 3. Usage: Sync key from ref with templating
  - secretRef:
      key: /source/credentials/password
    target:
      key: /target/example-3

    # Template defines how the secret will be synced to target store.
    # Either "rawData" or "data" should be specified, not both.
    template:
      rawData: '{{ .Data }}'   # Save as raw (accepts multiline string)
      data:                    # Save as map (accepts nested values)
        example: '{{ .Data }}'

  ## 4. Usage: Sync all keys from query with templating
  - secretQuery:
      path: /source/credentials
      key:
        regexp: .*
    target:
      keyPrefix: /target/example-4/
    template:
      rawData: 'SECRET-PREFIX-{{ .Data }}'

  ## 5. Usage: Sync single key from query with templating
  - secretQuery:
      path: /source/credentials/query-data/
      key:
        regexp: (username|password)
    target:
      key: /target/example-5

    template:
      data:
        user: '{{ .Data.username }}'
        pass: '{{ .Data.password }}'

  ## 6. Usage: Sync single key from multiple sources with templating
  - secretSources:
      - name: username # Username mapping, available as ".Data.username"
        secretRef:
          key: /source/credentials/username

      - name: password # Password mapping, available as ".Data.password"
        secretRef:
          key: /source/credentials/password

      - name: dynamic_query # Query mapping, available as "Data.dynamic_query.<key>"
        secretQuery:
          path: /source/credentials
          key:
            regexp: .*

    target:
      key: /target/example-6

    template:
      data:
        username: '{{ .Data.username }}'
        password: '{{ .Data.password }}'
        userpass: '{{ .Data.dynamic_query.username }}/{{ .Data.dynamic_query.password }}'
```

#### Perform sync
```bash
secret-sync --source path/to/vault-source.yaml \
            --target path/to/vault-target.yaml \
            --sync path/to/sync-job.yaml
# Use --schedule "@every 1m" to override sync job file config.
```

</details>

### Docs
Check documentation and example usage at [DOCS](docs/).
