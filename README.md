# Secret Sync


[![go.dev - references](https://img.shields.io/badge/go.dev-references-047897)](https://pkg.go.dev/github.com/bank-vaults/secret-sync)
[![Go Report Card](https://goreportcard.com/badge/github.com/bank-vaults/secret-sync?style=flat-square)](https://goreportcard.com/report/github.com/bank-vaults/secret-sync)
[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbank-vaults%2Fsecret-sync.svg?type=shield)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbank-vaults%2Fsecret-sync?ref=badge_shield)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/bank-vaults/secret-sync/ci.yaml?branch=main&style=flat-square)](https://github.com/bank-vaults/secret-sync/actions/workflows/ci.yaml?query=workflow%3ACI)

Secret Sync exposes a generic way to interact with external secret storage systems like
[HashiCorp Vault](https://www.vaultproject.io/), [AWS Secrets Manager](https://aws.amazon.com/secrets-manager/), [Google Secrets Manager](https://cloud.google.com/secret-manager), [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/), and others.
It then builds on it to provide a way to explicitly define how secrets should be synchronized between these stores using a set of API models and custom resources.

This name was chosen in a rush, we are open to naming suggestions 😄

> [!IMPORTANT]
> This is an **early alpha version** and breaking changes are expected.
> As such, it is not recommended for usage in production.
> We are actively working on expanding the list of supported stores and consolidating our APIs.
>
> You can support us with your feedback, bug reports, and feature requests.

## Features

- Seamless integration with a variety of secret storage systems (stores)
- User-friendly API for defining synchronization actions on a secret-level
- Advanced templating capabilities for defining and transforming secrets
- Facilitate interaction between stores using Golang packages or the CLI


| **Supported store**                | **Status** |
|------------------------------------|------------|
| [HashiCorp's Vault](#secret-store) | _alpha_    |
| [Local](#secret-store)             | _alpha_    |

Check details about upcoming features by visiting the [project issue](https://github.com/bank-vaults/secret-sync/issues) board.

## Goal

* Provide safe and simple way to consume secrets
* Common API regardless of the secret store backend
* Explicit control over the secret synchronization process

> Consider a situation where Dev teams need access to secrets from different environments.
> Ops teams can provide access to secrets in the form of a sandboxed environment (e.g. new Vault instance) synced only with secrets Devs require; all in GitOps way.

## Getting Started

To get familiarized, we will show how you can use these tools to answer two questions:

- How do I sync secrets from one store to another?
- How do I consume secrets to bootstrap my configs?

To answer the first question, we shall create some database secrets and synchronize them into Vault.<br>
For the second question, we will use some secrets from Vault to create an access file for an application.

You can find complete examples and instructions in the [EXAMPLE](EXAMPLE.md) file.

## Documentation

### Secret Store

Secret store defines the actual secret store that will be used for API requests.
In API requests, a secret store can be either a _source_ where the secrets are fetched from or a _target_ where
the requested secrets are synced into.

```yaml
# Defines a specific store to use. Only one store can be specified.
secretsStore:
  # Each store has a unique name and associated specs.
  storeName: storeSpec
```

You can find all the Secret Store specifications in [pkg/apis/v1alpha1/secretstore.go](pkg/apis/v1alpha1/secretstore.go)

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
<summary>Store Spec: <b>Local</b></summary>

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
Each sync action defines a specific mode of operation depending on its specifications.
You can use this as a reference point to create a more complete sync process based on the given requirements.

```yaml
# Used to configure the schedule for synchronization. Optional, runs only once if empty.
# The schedule is in Cron format, see https://en.wikipedia.org/wiki/Cron
schedule: "@daily"

# Defines sync actions, i.e. how and what will be synced. Requires at least one.
sync:
  - actionSpec
  - actionSpec
```

You can find all the Sync Plan specifications in [pkg/apis/v1alpha1/syncjob_types.go](pkg/apis/v1alpha1/syncjob_types.go)

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

### Running the synchronization

The CLI tool provides a way to run secret synchronization between secret stores.
It requires three things:
- Path to _source store_ config file via `--source` flag
- Path to _target store_ config file via `--target` flag
- Path to _sync plan_ config file via `--plan` flag

Note that only YAML configuration files are supported.
You can also provide optional params for CRON schedule to periodically sync secrets via `--schedule` flag.
All sync actions are indexed in logs based on their order in the sync plan config file.

You can also use [pkg/storesync](pkg/storesync) package to run secret synchronization plan natively from Golang.
This is how the CLI works as well.

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

## Useful links

- [Contributing guide](https://bank-vaults.dev/docs/contributing/)
- [Security procedures](https://bank-vaults.dev/docs/security/)
- [Code of Conduct](https://bank-vaults.dev/docs/code-of-conduct/)
- Email: [team@bank-vaults.dev](mailto:team@bank-vaults.dev)

## License

The project is licensed under the [Apache 2.0 License](https://github.com/bank-vaults/secret-sync/blob/master/LICENSE).

[![FOSSA Status](https://app.fossa.com/api/projects/git%2Bgithub.com%2Fbank-vaults%2Fsecret-sync.svg?type=large)](https://app.fossa.com/projects/git%2Bgithub.com%2Fbank-vaults%2Fsecret-sync?ref=badge_large)
