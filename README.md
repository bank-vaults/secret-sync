## Secret Sync

Perform secret synchronization between secret stores (e.g. Hashicorp Vault to AWS Secret Manager) in a configurable and explicit manner.
This tool is intended to be used by both developer and operation teams.

> [!WARNING]  
> This is an early alpha version and there will be changes made to the API. You can support us with your feedback.

### Supported secret stores
<details>
<summary>- Hashicorp Vault</summary>

Uses Hashicorp Vault as a secret store service.
Use the following configuration to use `DirProvider` as a secret store.
```yaml
vault:
    address: "http://0.0.0.0:8200"
    storePath: "secret"
    role: ""
    authPath: "userpass"
    tokenPath: ""
    token: "root" # unseal token
```
</details>

<details>
<summary>- Local directory</summary>

The folder defines a secret store while the secrets are plain unencrypted files within that directory.
Use the following configuration to use `DirProvider` as a secret store.
```yaml
local:
  storePath: "path/to/store-root"
```
</details>

### Quick start


#### 1. Setup secret stores
Create a Local provider to serve as a source store (where the secrets will be synced from)
```bash
# Create local dir
mkdir /tmp/source-store

# Create provider config file
cat <<EOF > /tmp/source-provider.yaml
local:
  storePath: "/tmp/source-store"
EOF
```

Create a Vault provider to serve as a target store (where the secrets will be synced to)
```bash
# Deploy a Vault instance
docker compose -f e2e/vault/docker-compose.yaml up -d

# Create provider config file
cat <<EOF > /tmp/target-provider.yaml
vault:
    address: "http://0.0.0.0:8200"
    storePath: "secret" # Vault KV store
    authPath: "userpass"
    token: "root"
EOF
```

### 2. Create secrets on source
Create some secrets in source store (which will be synced to target store in the next step).
```bash
echo "secret-1" > /tmp/source-store/secret-1
echo "secret-2" > /tmp/source-store/secret-2
echo "secret-2" > /tmp/source-store/secret-3
```

### 3. Create sync plan
We will define how we want to perform secret synchronization between source and target stores.
More examples on how to create a sync plan is documented in chapter (Sync Plan)[#sync-plan].
```bash
cat <<EOF > /tmp/sync-plan.yaml
sync:
  - secretQuery:
      path: /
      key:
        regexp: .*
EOF
```
### 4. Sync secrets
```bash
secret-sync --source /tmp/source-provider.yaml \
            --target /tmp/target-provider.yaml \
            --sync /tmp/sync-plan.yaml
```
This should output something like:
```bash
{"level":"info","msg":"Successfully synced plan item = 0 for key /secret-1"}
{"level":"info","msg":"Successfully synced plan item = 0 for key /secret-2"}
{"level":"info","msg":"Successfully synced plan item = 0 for key /secret-3"}
{"level":"info","msg":"Synced 3 out of total 3 keys"
```
You can now navigate to the requested Vault secret store at [localhost:8200/ui/vault/secrets/secret/list](http://localhost:8200/ui/vault/secrets/secret/list) and login with `root` as token.
You should now see `secret-1`, `secret-2`, and `secret-3`.
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
    flatten: true
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
