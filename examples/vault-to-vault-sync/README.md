# Getting Started

To get familiarized, we will show how you can use the Secret Sync tool to synchronize secrets from a vault instance to another.

## Step 1: Prepare environment

You will need the following tools to continue:

- Docker compose
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

### step 2: Prepare vault instances

```bash
# Create a directory for the examples
mkdir -p /tmp/example/

# Deploy the two vault instances
make up

# Enable kv-v2 secrets engine at: serviceA and serviceB
docker exec -it secret-sync-vault-1 vault secrets enable -path=serviceA kv-v2
docker exec -it secret-sync-vault-2 vault secrets enable -path=serviceB kv-v2

# Create approles at: serviceA and serviceB
docker exec -it secret-sync-vault-1 vault auth enable -path=serviceA approle
docker exec -it secret-sync-vault-2 vault auth enable -path=serviceB approle

# Create the secrets in the source Vault
docker exec -it secret-sync-vault-1 vault kv put serviceA/database/config username=user password=pass
```

### Step 3: Define secret stores

#### Vault store

Create the config files to use as *Vault secret stores*.

```bash
# Create the first Vault store config file
cat <<EOF > /tmp/example/vault-store.yml
secretsStore:
  vault:
    address: "http://127.0.0.1:8200"
    storePath: "serviceA"
    authPath: "serviceA"
    token: "227e1cce-6bf7-30bb-2d2a-acc854318caf"
EOF

# Create the second Vault store config file
cat <<EOF > /tmp/example/vault-store-2.yml
secretsStore:
  vault:
    address: "http://127.0.0.1:8201"
    storePath: "serviceB"
    authPath: "serviceB"
    token: "227e1cce-6bf7-30bb-2d2a-acc854318caf"
EOF
```

### Step 4: Define a sync plan

Define a sync plan for the secrets:

- `database/config/username`
- `database/config/password`

These secrets will be synced from the first vault to the second vault secret store.

```bash
cat <<EOF > /tmp/example/db-secrets-sync-from-vault-to-vault.yml
sync:
  - secretRef:
      key: database/config/username
    target:
      key: database/config/username/username-synced

  - secretRef:
      key: database/config/password
    target:
      key: database/config/password/password-synced
EOF
```

### Step 5: Perform sync

Secret synchronization is performed using the CLI by executing the sync plan between source and target secret stores.

#### Synchronize Database Secrets

To synchronize the database secrets from the first Vault to the second one, run:

```bash
secret-sync sync \
--source "/tmp/example/vault-store.yml" \
--target "/tmp/example/vault-store-2.yml" \
--syncjob "/tmp/example/db-secrets-sync-from-vault-to-vault.yml"
```

If successful, your output should contain something like:

```json
{"level":"INFO","msg":"Successfully synced action","app":"secret-sync","id":0,"key":"database/config/username/username-synced"}
{"level":"INFO","msg":"Successfully synced action","app":"secret-sync","id":1,"key":"database/config/password/password-synced"}
{"level":"INFO","msg":"Synced 2 out of total 2 keys","app":"secret-sync"}
```

You can also retrieve the secrets from the second Vault instance to verify that the synchronization was successful.

```bash
docker exec -it secret-sync-vault-2 vault kv get -mount="serviceB" "database/config/username"
docker exec -it secret-sync-vault-2 vault kv get -mount="serviceB" "database/config/password"
```

### Step 6: Cleanup

```bash
# Destroy Vault instances
make down

# Remove example assets
rm -rd /tmp/example
rm -rd /tmp/secret-sync
```
