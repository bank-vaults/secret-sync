# Secret Sync

Perform secret synchronization between secret stores (e.g. Hashicorp Vault to AWS Secret Manager) in a configurable and explicit manner.
The goal of this project is to make usage and management of secrets as simple as possible for both developer and operation teams.

> [!WARNING]
> This is an early alpha version and there will be changes made to the API. You can support us with your feedback.

### Supported secret stores
The list of providers is rather small since the project is still in its early stage.
Our goal is to gradually over time expand the list of supported providers,
as well as consolidate APIs across and for providers.

<details>
<summary><b>HashiCorp Vault*</b></summary>

The following configuration selects [HashiCorp Vault](https://www.vaultproject.io/) as a secret store.
Find example usage of this provider in Quick Start section.
```yaml
secretsStore:
  vault:
    address: "<Vault API endpoint>"
    storePath: "<Vault path to secrets store>"
    authPath: "<Vault path to auth role>"
    role: "<Auth role>"
    tokenPath: "<Local path to Vault token>"
    token: "<Vault token>"
```
_*Vault needs to be unsealed_.
</details>

<details>
<summary><b>Local directory</b></summary>

Use this configuration to specify local directory as a secret store.
Secrets are represented as unencrypted files within that directory,
where filenames define secret keys and file contents the secret values.
Find example usage of this provider in Quick Start section.
```yaml
secretsStore:
  local:
    storePath: "path/to/local-dir"
```
</details>

## Goal

The goal of _secret sync_ is to try to solve some common issues related to secret usage and management.
More specifically:
1. How to synchronize secrets from one service to another?
2. How to explicitly define the synchronization rules for secrets?
2. How to enable secret templating and bootstrapping capabilities?

Consider a situation where Dev teams need access to secrets from a specific environment.
To enable tenancy and GitOps, Ops teams can create sandboxed environments (e.g. new Vault instance) and synchronize
into it only the secrets Dev teams can access or might require.
In turn, Dev teams can easily access and use secrets in the same configurable manner.

---

## Quick start

In this example, we will show how you can synchronize specific secrets from a local directory to Vault instance and vice versa.

#### Prepare environment
Requirements:
- Git
- Docker
- Golang 1.21

Clone and build secret sync:
```bash
git clone https://github.com/bank-vaults/secret-sync.git
cd secret-sync
make build
```

#### Define secret stores
Use local directory as the source secret store, ie. where the secrets will be synced from.
```bash
# Create dir to use for source secret store
mkdir /tmp/source-store

# Create source provider config file, use local dir
cat <<EOF > /tmp/source-provider.yml
secretsStore:
  local:
    storePath: "/tmp/source-store"
EOF
```

Use Vault instance as the target secret store, ie. where the secrets will be synced to.
```bash
# Deploy a Vault instance for target secret store
docker compose -f dev/vault/docker-compose.yml up -d

# Create target provider config file, use Vault
cat <<EOF > /tmp/target-provider.yml
secretsStore:
  vault:
    address: "http://0.0.0.0:8200"
    storePath: "secret/" # Vault store path
    authPath: "userpass"
    token: "root"
EOF
```

### Create secrets to sync
Create some secrets in the source store, ie. secrets to sync.
```bash
echo "VerySecretData1" > /tmp/source-store/secret-1
echo "VerySecretData2" > /tmp/source-store/secret-2
echo "VerySecretData3" > /tmp/source-store/secret-3
```

### Define sync plan
Create a plan to define the sync steps between secret stores, ie. how to sync secrets.
Examples on how to specify a more complex sync plan are documented in chapter [Sync Plan](#sync-plan).

```bash
cat <<EOF > /tmp/sync-plan.yml
sync:
  # We want to sync all secrets that starts with "secret-" from source store path "/"
  # to the same path on target. Keys on target will be synced with "synced-" prefix.
  # It is possible to specify multiple sync items, as well as to use templating,
  # for a more complex sync plan.
  # Sync items are indexed in logs based on their order in this config file.
  - secretQuery:
      path: /
      key:
        regexp: secret-.*
      target:
        keyPrefix: synced-
EOF
```

### Sync secrets
You are now ready to perform secret synchronization between secret stores.
Use `secret-sync` to execute sync plan between source and target secret stores.
```bash
./build/secret-sync --source "/tmp/source-provider.yml" --target "/tmp/target-provider.yml" --sync "/tmp/sync-plan.yml"
```

Wh should output something like:
```bash
{"level":"info","msg":"Successfully synced plan item = 0 for key /secret-1"}
{"level":"info","msg":"Successfully synced plan item = 0 for key /secret-2"}
{"level":"info","msg":"Successfully synced plan item = 0 for key /secret-3"}
{"level":"info","msg":"Synced 3 out of total 3 keys"
```
To validate, navigate to the target secret store on the local Vault instance at [localhost:8200/ui/vault/secrets/secret/list](http://localhost:8200/ui/vault/secrets/secret/list) and login with `root` as token.
You should now see `synced-secret-1`, `synced-secret-2`, and `synced-secret-3` keys.
The values are base64 encoded.

## Advanced usage

### Sync Plan

<details>
<summary>More advanced information on Sync Plan</summary>

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

---

Additional info here
