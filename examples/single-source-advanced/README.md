## Getting Started

To get familiarized, we will show how you can use Secret Sync tool to cover two scenarios:

- **Case 1**: Synchronize secrets from one secret store to another.
  - We will use our local file store as a source of truth to synchronize some database secrets into Vault.
- **Case 2**: Consume secrets to bootstrap an application.
  - We will use Vault instance to fetch database secrets to our local store in a form of a configuration file for an application.

*Note:* The same logic applies to any other combination of secret stores.

### Step 1: Prepare environment

You will need the following tools to continue:
- Git
- Makefile
- Golang `>= 1.21`

To set up the environment, you can build from source:

```bash
git clone https://github.com/bank-vaults/secret-sync.git /tmp/secret-sync
cd /tmp/secret-sync
make build
alias secret-sync="/tmp/secret-sync/build/secret-sync"
```

Alternatively, you can also use only Docker:

```bash
alias secret-sync="docker run --rm -v /tmp:/tmp ghcr.io/bank-vaults/secret-sync:v0.1.0 secret-sync"
```

### Step 2: Define secret stores

#### Local store

Create a directory and a config file to use as the _local secret store_.

```bash
# Create local store directory
mkdir -p /tmp/example/local-store

# Create local store config file
cat <<EOF > /tmp/example/local-store.yml
secretsStore:
  local:
    storePath: "/tmp/example/local-store"
EOF
```

#### Vault store

Deploy Vault and create config file to use as the _Vault secret store_.

```bash
# Deploy a Vault instance
docker compose -f dev/vault/docker-compose.yml up -d

# Create Vault store config file
cat <<EOF > /tmp/example/vault-store.yml
secretsStore:
  vault:
    address: "http://0.0.0.0:8200"
    storePath: "secret/"
    authPath: "userpass"
    token: "root"
EOF
```

### Step 3: Define sync plans

#### Database secrets

Define a sync plan for `db-host`, `db-user`, `db-pass` secrets. These secrets will be synced from our local to Vault secret store.

```bash
cat <<EOF > /tmp/example/db-secrets-sync.yml
sync:
  - secretQuery:
      path: /
      key:
        regexp: db-(host|user|pass)
EOF
```

#### Application access secret

Define a sync plan for app-specific secret `app-access-config` created from various other secrets (e.g. database). This secret will be synced from Vault to our local secret store (as a file). It can also be synced against the same store to refresh the secret.

```bash
cat <<EOF > /tmp/example/app-access-config-sync.yml
sync:
  - secretSources:
      - name: selector
        secretQuery:
          path: /
          key:
            regexp: db-(host|user|pass)
    target:
      key: app-access-config
    template:
      data:
        appID: "12345"
        # ...some additional secrets for the given app...

        # Secrets fetched from Vault will be encoded, we need to decode
        hostname: "{{ .Data.selector.dbHost | base64dec }}"
        username: "{{ .Data.selector.dbUser | base64dec }}"
        password: "{{ .Data.selector.dbPass | base64dec }}"
EOF
```

### Step 4: Create database secrets

Create database access secrets in our local secret store.

```bash
echo -n "very-secret-hostname" > /tmp/example/local-store/db-host
echo -n "very-secret-username" > /tmp/example/local-store/db-user
echo -n "very-secret-password" > /tmp/example/local-store/db-pass
```

### Step 5: Perform sync

Secret synchronization is performed using the CLI by executing the sync plan between source and target secret stores.

#### Database secrets

To synchronize database secrets from our local to Vault secret store, run:

```bash
secret-sync --source "/tmp/example/local-store.yml" --target "/tmp/example/vault-store.yml" --sync "/tmp/example/db-secrets-sync.yml"
```

If successful, your output should contain something like:

```json
{"level":"info","msg":"Successfully synced action = 0 for key /db-user"}
{"level":"info","msg":"Successfully synced action = 0 for key /db-pass"}
{"level":"info","msg":"Successfully synced action = 0 for key /db-host"}
{"level":"info","msg":"Synced 3 out of total 3 keys"}
```

You can also navigate to the local Vault instance and verify these secrets.

#### Application access secret

To synchronize application access secret from Vault to our local secret store, run:

```bash
secret-sync --target "/tmp/example/local-store.yml" --source "/tmp/example/vault-store.yml" --sync "/tmp/example/app-access-config-sync.yml"
```

If successful, beside logs, you should also be able to find the app access secret via:

```bash
cat /tmp/example/local-store/app-access-config
{"appID":"12345","hostname":"very-secret-hostname","password":"very-secret-password","username":"very-secret-username"}
```


### Step 6: Cleanup

```bash
# Destroy Vault instance
docker compose -f dev/vault/docker-compose.yml down

# Remove example files
rm -rf /tmp/example
```
