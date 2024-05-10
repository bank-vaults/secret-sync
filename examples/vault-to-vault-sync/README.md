# Getting Started

To get familiarized, we will show how you can use the Secret Sync tool to synchronize secrets from a vault instance to another.

## Step 1: Prepare environment

You will need the following tools to continue:

- Docker compose
- Makefile
- Golang `>= 1.21`

To set up the environment, you can build from source:

```bash
make build

alias secret-sync="build/secret-sync"
```

Alternatively, you can also use only Docker:

```bash
alias secret-sync="docker run --rm -v /tmp:/tmp ghcr.io/bank-vaults/secret-sync:v0.1.0 secret-sync"
```

### step 2: Prepare vault instances

```bash
# Create a directory for the examples
mkdir -p tmp/example/

# Deploy the two vault instances
docker compose -f dev/vault/docker-compose2.yml up -d

# Enable kv-v2 secrets engine at: internal
docker exec -it vault-1 vault secrets enable -path=internal kv-v2
docker exec -it vault-2 vault secrets enable -path=internal kv-v2

# Create approles at: internal-app
docker exec -it vault-1 vault auth enable -path=internal-app approle
docker exec -it vault-2 vault auth enable -path=internal-app approle

# Create the secrets in the source Vault
docker exec -it vault-1 vault kv put internal/database/config username=user password=pass
```

### Step 3: Define secret stores

#### Vault store

Create the config files to use as *Vault secret stores*.

```bash
# Create the first Vault store config file
cat <<EOF > tmp/example/vault-store.yml
secretsStore:
  vault:
    address: "http://127.0.0.1:8200"
    storePath: "internal"
    authPath: "internal-app"
    token: "root"
EOF

# Create the second Vault store config file
cat <<EOF > tmp/example/vault-store-2.yml
secretsStore:
  vault:
    address: "http://127.0.0.1:8201"
    storePath: "internal"
    authPath: "internal-app"
    token: "root"
EOF
```

### Step 4: Define a sync plan

Define a sync plan for the secrets:

- `database/config/username`
- `database/config/password`

These secrets will be synced from the first vault to the second vault secret store.

```bash
cat <<EOF > tmp/example/db-secrets-sync-from-vault-to-vault.yml
sync:
  - secretRef:
      key: database/config/username
    target:
      key: database/config/username/username

  - secretRef:
      key: database/config/password
    target:
      key: database/config/password/password
EOF
```

### Step 5: Perform sync

Secret synchronization is performed using the CLI by executing the sync plan between source and target secret stores.

#### Synchronize Database Secrets

To synchronize the database secrets from the first Vault to the second one, run:

```bash
secret-sync --source "tmp/example/vault-store.yml" --target "tmp/example/vault-store-2.yml" --sync "tmp/example/db-secrets-sync-from-vault-to-vault.yml"
```

If successful, your output should contain something like:

```json
Synced 2 out of total 2 keys
```

You can also retrieve the secrets from the second vault instances to verify these secrets.

```bash
Docker exec -it vault-2 vault kv get -mount="internal" "database/config/username"
Docker exec -it vault-2 vault kv get -mount="internal" "database/config/password"
```

### Step 6: Cleanup

```bash
# Destroy Vault instances
docker compose -f dev/vault/docker-compose2.yml down

# Remove tmp directory
rm -rd tmp/
```
