# Getting Started

To get familiarized, we will show how you can use Secret Sync tool to cover two scenarios:

- **Case 1**: Synchronize secrets from one secret store to another.
  - We will use our local file store as the source of truth to synchronize some database secrets into Vault.
- **Case 2**: Consume secrets to bootstrap an application.
  - We will use a Vault instance to fetch database secrets to our local store in the form of a configuration file for an application.

*Note:* The same logic applies to any other combination of secret stores.

## Step 1: Prepare environment

You will need the following tools to continue:

- Docker compose
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
alias secret-sync="docker run --rm -v /tmp:/tmp ghcr.io/bank-vaults/secret-sync:latest secret-sync"
```

### Step 2: Define secret stores

#### Local store

Create a directory and a config file to use as the *local secret store*.

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

Deploy Vault and create config file to use as the *Vault secret store*.

```bash
# Deploy a Vault instance
make up

# Create an approle at: serviceA
docker exec -it secret-sync-vault-1 vault auth enable -path=serviceA approle

# Create Vault store config file
cat <<EOF > /tmp/example/vault-store.yml
secretsStore:
  vault:
    address: "http://0.0.0.0:8200"
    storePath: "secret"
    authPath: "serviceA"
    token: "227e1cce-6bf7-30bb-2d2a-acc854318caf"
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

        hostname: "{{ .Data.selector.dbHost }}"
        username: "{{ .Data.selector.dbUser }}"
        password: "{{ .Data.selector.dbPass }}"
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

#### Synchronize database secrets

To synchronize database secrets from our local to Vault secret store, run:

```bash
secret-sync \
--source "/tmp/example/local-store.yml" \
--target "/tmp/example/vault-store.yml" \
--sync "/tmp/example/db-secrets-sync.yml"
```

If successful, your output should contain something like:

```json
{"level":"INFO","msg":"Successfully synced action","app":"secret-sync","id":0,"key":"/db-pass"}
{"level":"INFO","msg":"Successfully synced action","app":"secret-sync","id":0,"key":"/db-host"}
{"level":"INFO","msg":"Successfully synced action","app":"secret-sync","id":0,"key":"/db-user"}
{"level":"INFO","msg":"Synced 3 out of total 3 keys","app":"secret-sync"}
```

You can also navigate to the local Vault instance and verify these secrets.

```bash
docker exec -it secret-sync-vault-1 vault kv get -mount="secret" "db-user"
docker exec -it secret-sync-vault-1 vault kv get -mount="secret" "db-pass"
docker exec -it secret-sync-vault-1 vault kv get -mount="secret" "db-host"
```

#### Synchronize application access secret

To synchronize application access secret from Vault to our local secret store, run:

```bash
secret-sync \
--target "/tmp/example/local-store.yml" \
--source "/tmp/example/vault-store.yml" \
--sync "/tmp/example/app-access-config-sync.yml"
```

If successful, besides logs:

```json
{"level":"INFO","msg":"Successfully synced action","app":"secret-sync","id":0,"key":"app-access-config"}
{"level":"INFO","msg":"Synced 1 out of total 1 keys","app":"secret-sync"}
```

You should also be able to find the secrets at the target store path:

```bash
cat /tmp/example/local-store/app-access-config
```

Output:

```json
{"appID":"12345","hostname":"very-secret-hostname","password":"very-secret-password","username":"very-secret-username"}
```

### Step 6: Cleanup

```bash
# Destroy Vault instance
make down

# Remove example assets
rm -rd /tmp/example
rm -rd /tmp/secret-sync
```
