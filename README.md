# Secret Sync

[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/bank-vaults/secret-sync/ci.yaml?branch=main&style=flat-square)](https://github.com/bank-vaults/secret-sync/actions/workflows/ci.yaml?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/bank-vaults/secret-sync?style=flat-square)](https://goreportcard.com/report/github.com/bank-vaults/secret-sync)

Perform secret synchronization between secret stores (e.g. Hashicorp Vault to AWS Secret Manager) in a configurable manner.
Enable seamless interoperability between different secret stores and make use of explicit API to control when, what, and how to synchronize.

| **Store**                          | **Status** |
|------------------------------------|------------|
| [HashiCorp's Vault](#secret-store) | _alpha_    |
| [Local Directory](#secret-store)   | _alpha_    |

> [!IMPORTANT]
> This is an **early alpha version** and breaking changes are expected.
> As such, it is not recommended for usage in production.
> We are actively working on expanding the list of supported stores and consolidating our APIs.
>
> You can support us with your feedback, bug reports, and feature requests.

## Goal

Secret Sync tries to tackle common issues related to secret usage and management lifecycle.
Specifically, it aims to:
* _Allow unified secret exchange between different stores_
* _Define and perform synchronization steps clearly and explicitly_
* _Enable safe and simple consumption of secrets_

> Consider a situation where Dev teams need access to secrets from different environments.
> Ops teams can provide access to secrets in the form of a sandboxed environment (e.g. new Vault instance) synced only with secrets Devs require; all in GitOps way.

## Getting Started

To get familiarized, we will show how you can use Secret Sync to answer two questions:

- How do I sync secrets from one store to another?
- How do I consume secrets to bootstrap my configs?

To answer the first question, we shall create some database secrets and synchronize them into Vault.<br>
For the second question, we will use some secrets from Vault to create an access file for an application.

### 1. Prepare environment

You will need the following tools to continue:
- Docker
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
alias secret-sync="docker run --net=host --user $(id -u):$(id -g) --rm -v /tmp/example:/tmp/example ghcr.io/bank-vaults/secret-sync:latest secret-sync"
```

### 2. Define secret stores

Documentation and examples on how to use different secret stores can be found in chapter [Secret Store](#secret-store).

#### 2.1. Local store
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

#### 2.2. Vault store
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

### 3. Define sync plans
Documentation and examples on how to create a more extensive sync plan can be found in chapter [Sync Plan](#sync-plan).

#### 3.1. Database secrets
Define a sync plan for `db-host`, `db-user`, `db-pass` secrets.
These secrets will be synced from our local to Vault secret store.

```bash
cat <<EOF > /tmp/example/db-secrets-sync.yml
sync:
  - secretQuery:
      path: /
      key:
        regexp: db-(host|user|pass)
EOF
```

#### 3.1. Application access secret
Define a sync plan for app-specific secret `app-access-config` created from various other secrets (e.g. database).
This secret will be synced from Vault to our local secret store (as a file).
It can also be synced against the same store to refresh the secret.

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

### 4. Create database secrets

Create database access secrets in our local secret store.
```bash
echo -n "very-secret-hostname" > /tmp/example/local-store/db-host
echo -n "very-secret-username" > /tmp/example/local-store/db-user
echo -n "very-secret-password" > /tmp/example/local-store/db-pass
```

### 5. Perform sync

Secret synchronization is performed using the [CLI](#syncing-with-cli) by executing the sync plan between source and target secret stores.

#### 5.1. Database secrets

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

#### 5.2. Application access secret

To synchronize application access secret from Vault to our local secret store, run:

```bash
secret-sync --target "/tmp/example/local-store.yml" --source "/tmp/example/vault-store.yml" --sync "/tmp/example/app-access-config-sync.yml"
```

If successful, beside logs, you should also be able to find the app access secret via:
```bash
cat /tmp/example/local-store/app-access-config
# {"appID":"12345","hostname":"very-secret-hostname","password":"very-secret-password","username":"very-secret-username"}
```


### 6. Cleanup

```bash
# Destroy Vault instance
docker compose -f dev/vault/docker-compose.yml down

# Remove example files
rm -rf /tmp/example
```

## Documentation

### Secret Store

Secret store defines the actual secret store that will be used for API requests.
In API requests, a secret store can be either a **source** where the secrets are fetched from or a **target** where
the requested secrets are synced into.
```yaml
# Defines a specific store to use. Only one store can be specified.
secretsStore:
  # Each store has a unique name and associated specs.
  storeName: storeSpec
```

<details>
<summary>Store Spec: <b>HashiCorp's Vault*</b></summary>

#### Specs

The following configuration selects [HashiCorp's Vault](https://www.vaultproject.io/) as a secret store.
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
<summary>Store Spec: <b>Local directory</b></summary>

#### Specs

Use this configuration to specify a local directory as a secret store.
Secrets are represented as unencrypted files within that directory,
where filenames define secret keys and file contents the secret values.
This store is useful for local secret consumption.
```yaml
secretsStore:
  local:
    storePath: "path/to/local-dir"
```
</details>

### Sync Plan

Sync plan consists of general configurations and a list of sync actions that enable the selection, transformation, and synchronization of secrets from source to target stores.

```yaml
# Used to configure the schedule for synchronization. Optional, runs only once if empty.
# The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
schedule: "@daily"

# Defines sync actions, i.e. how and what will be synced. Requires at least one.
sync:
  - actionSpec
  - actionSpec
```

Each sync action specifies one of four modes of operation depending on the specifications.
You can use this as a reference point to create a more complete sync process based on the given requirements.

<details>
<summary>Action Spec: <b>Synchronize a secret from reference</b></summary>

#### Specs

```yaml
sync:
    # Specify which secret to fetch from source. Required.
  - secretRef:
      key: /path/in/source-store/key

    # Specify where the secrets will be synced to on target. Optional.
    # If empty, will be the same as "secretRef.key".
    target:
      key: /path/in/target-store/key

    # Template defines how to transform secret before syncing to target. Optional.
    # If set, either "template.rawData" or "template.data" must be specified.
    #
    # The template will be executed once to create a value to sync to "target.key".
    # The value of the "secretRef.key" secret can be accessed via {{ .Data }}.
    template:
      rawData: '{{ .Data }}'  # save either as a (multiline) string
      data:                   # or as a map
        secretPassword: '{{ .Data }}'
```

#### Example

Synchronize a single `/tenant-1/db-username` from the source store to `/remote-db-username` on the target store.
```yaml
sync:
- secretRef:
    key: /tenant-1/db-username
  target:
    key: /remote-db-username
```

</details>

<details>
<summary>Action Spec: <b>Synchronize multiple secrets from a query</b></summary>

#### Specs

```yaml
sync:
    # Specify query for secrets to fetch from source. Required.
  - secretQuery:
      path: /path/in/source-store
      key:
        regexp: some-key-prefix-.*

    # Specify where the secrets will be synced to on target. Optional.
    # > If set, every query matching secret will be synced under
    #     key = "{target.keyPrefix}{match.GetName()}"
    # > If empty, every query matching secret will be synced under
    #     key = "{secretQuery.path}/{match.GetName()}".
    target:
      keyPrefix: /path/in/target-store/

    # Template defines how to transform secret before syncing to target. Optional.
    # If set, either "template.rawData" or "template.data" must be specified.
    #
    # This template will be executed for every query matching secret to create a secret
    # which will be synced to "target".
    # The value of (current) query secret can be accessed via {{ .Data }}.
    template:
      rawData: '{{ .Data }}'  # save either as a (multiline) string
      data:                   # or as a map
        secretPassword: '{{ .Data }}'
```

#### Example

Synchronize all secrets that match `/tenant-1/db-*` regex from the source store to `/remote-<key>` on the target store.
```yaml
sync:
- secretQuery:
    path: /tenant-1
    key:
      regexp: db-.*
  target:
    keyPrefix: /remote-
```

</details>

<details>
<summary>Action Spec: <b>Synchronize a secret from a query</b></summary>

#### Specs

```yaml
sync:
    # Specify query for secrets to fetch from source. Required.
  - secretQuery:
      path: /path/in/source-store
      key:
        regexp: some-key-prefix-.*

    # Indicate that you explicitly want to sync into a single key. Required.
    flatten: true

    # Specify where the secret will be synced to on target. Required.
    target:
      key: /path/in/target-store/key

    # Template defines how to transform secret before syncing to target. Optional.
    # If set, either "template.rawData" or "template.data" must be specified.
    #
    # The template will be executed once to create a value which will be synced to "target.key".
    # The value for each secret from the "secretQuery" is accessible in the template
    # via {{ .Data.<camelCasedQueryName> }}, for example {{ .Data.someKeyPrefix1 }}.
    template:
      rawData: '{{ .Data.someKeyPrefix1 }}' # save either as a (multiline) string
      data:                                 # or as a map
        secret: '{{ .Data.someKeyPrefix1 }}'
```

#### Example

Fetch secrets that match `/tenant-1/db-(username|password)` regex from source store and use them
to create a new (combined) db access secret on the target store.

```yaml
sync:
- secretQuery:
    path: /tenant-1
    key:
      regexp: db-(username|password)
  flatten: true
  target:
    key: /db/access
  template:
    data:
      type: "postgres"
      username: "{{ .Data.dbUsername }}"
      password: "{{ .Data.dbPassword }}"
```

</details>


<details>
<summary>Action Spec: <b>Synchronize a secret from multiple queries and references</b></summary>

#### Specs

```yaml
sync:
    # Specify (named) queries and references for secrets to fetch from source.
    # At least one sync action is required.
  - secretSources:
    - name: action-ref
      secretRef:
        key: /path/in/source-store/key
    - name: action-query
      secretQuery:
        path: /path/in/source-store
        key:
          regexp: some-key-prefix-.*

    # Specify where the secret will be synced to on target. Required.
    target:
      key: /path/in/target-store/key

    # Template defines how to transform secret before syncing to target. Optional.
    # If set, either "template.rawData" or "template.data" must be specified.
    #
    # The template will be executed once to create a value which will be synced to "target.key".
    # The value for each secret from the "secretSources" is accessible in the template via:
    # > Use {{ .Data.<camelCaseSourceName> }} for "action-ref" source.
    #   For example, use {{ .Data.actionRef }}
    # > Use {{ .Data.<camelCaseSourceName>.<camelCasedQueryName> }} for "action-query" source.
    #   For example {{ .Data.actionQuery }}
    template:
      rawData: '{{ .Data.actionRef }}'
      data:
        secret1: '{{ .Data.actionRef }}'
        secret2: '{{ .Data.actionQuery.someKeyPrefix1 }}'
```

#### Example

Fetch secrets that match `/db-(1|2)/(username|password)` regex from source store and use them
to create a new (combined) db access secret on the target store.

```yaml
sync:
  - secretSources:
      - name: db1
        secretRef:
          path: /db-1
          key:
            regexp: username|password
      - name: db2
        secretRef:
          path: /db-2
          key:
            regexp: username|password
    target:
      key: /dbs-combined
    template:
      data:
        db1_username: "{{ .Data.db1.username }}"
        db1_password: "{{ .Data.db1.password }}"
        db2_username: "{{ .Data.db2.username }}"
        db2_password: "{{ .Data.db2.password }}"
```

</details>

#### On Templating

Standard golang templating is supported for sync action items.
In addition, functions such as `base64dec` and `base64enc` for decoding/encoding and
`contains`, `hasPrefix`, `hasSuffix` for string manipulation are also supported.

### CLI

The CLI tool provides a way to run secret synchronization between secret stores.
It requires three things:
- Path to _source store_ config file via `--source` flag
- Path to _target store_ config file via `--target` flag
- Path to _sync plan_ config file via `--plan` flag

Note that only YAML configuration files are supported.
You can also provide optional params for CRON schedule to periodically sync secrets via `--schedule` flag.
All sync actions are indexed in logs based on their order in the sync plan config file.

## Development

**For an optimal developer experience, it is recommended to install [Nix](https://nixos.org/download.html) and [direnv](https://direnv.net/docs/installation.html).**

_Alternatively, install [Go](https://go.dev/dl/) on your computer then run `make deps` to install the rest of the dependencies._

Fetch required tools:
```shell
make deps
```

Build the CLI:

```shell
make build
```

Run the test suite:

```shell
make test
```

Run linters:

```shell
make lint # pass -j option to run them in parallel
```

Some linter violations can automatically be fixed:

```shell
make fmt
```

## Getting help

- For feature requests and bugs, file an [issue](https://github.com/bank-vaults/secret-sync/issues).
- For general discussion about both usage and development:
  - join the [#secret-sync](https://outshift.slack.com/messages/secret-sync) on the Outshift Slack
  - open a new [discussion](https://github.com/bank-vaults/secret-sync/discussions)

## License

The project is licensed under the [Apache 2.0 License](https://github.com/bank-vaults/secret-sync/blob/master/LICENSE).
