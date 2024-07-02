# API Documentation

## Secret Store

Secret Store defines the actual external secret storage systems that will be used for API requests.
In API requests, a secret store can be either a _source_ where the secrets are fetched from or a _target_ where
the requested secrets are synced into.

```yaml
# Defines a specific store to use. Only one store can be specified.
secretsStore:
  # Each store has a unique name and associated specs.
  storeName: storeSpec
```

<details>
<summary>Store Spec: <b>HashiCorp Vault*</b></summary>

### Specs

The following configuration selects [HashiCorp Vault](https://www.vaultproject.io/) as a secret store.

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
<summary>Store Spec: <b>Local Provider</b></summary>

### Specs

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

<details>
<summary>Action Spec: <b>Synchronize a secret from reference</b></summary>

### Specs

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

### Specs

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

### Example

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

### Specs

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

### Example

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

### Specs

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

### Example

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
- Path to _sync plan_ config file via `--syncjob` flag

Note that only YAML configuration files are supported.
You can also provide optional params for CRON schedule to periodically sync secrets via `--schedule` flag.
All sync actions are indexed in logs based on their order in the sync plan config file.

You can also use [pkg/storesync](https://pkg.go.dev/github.com/bank-vaults/secret-sync/pkg/storesync) package to run secret synchronization plan natively from Golang.
This is how the CLI works as well.
