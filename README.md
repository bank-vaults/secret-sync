# Secret Sync

[![Go Report Card](https://goreportcard.com/badge/github.com/bank-vaults/secret-sync?style=flat-square)](https://goreportcard.com/report/github.com/bank-vaults/secret-sync)
![Go Version](https://img.shields.io/badge/go%20version-%3E=1.21-61CFDD.svg?style=flat-square)
[![go.dev - references](https://pkg.go.dev/badge/mod/github.com/bank-vaults/secret-sync)](https://pkg.go.dev/mod/github.com/bank-vaults/secret-sync)
[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/bank-vaults/secret-sync/ci.yaml?branch=main&style=flat-square)](https://github.com/bank-vaults/secret-sync/actions/workflows/ci.yaml?query=workflow%3ACI)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/bank-vaults/secret-sync/badge?style=flat-square)](https://api.securityscorecards.dev/projects/github.com/bank-vaults/secret-sync)
[![OpenSSF Best Practices](https://www.bestpractices.dev/projects/8055/badge)](https://www.bestpractices.dev/projects/8055)

**Secret Sync** exposes a generic way to interact with external secret storage systems like [HashiCorp Vault](https://www.vaultproject.io/) and provides a set of API models to interact and orchestrate the synchronization of secrets between them.

> [!IMPORTANT]
> This is an **early alpha version** and breaking changes are expected.
> As such, it is not recommended for usage in production.
> We are actively working on expanding the list of supported stores and consolidating our APIs.
>
> You can support us with your feedback, bug reports, and feature requests.

## Features

- Simple integration with a variety of secret storage systems
- User-friendly API to facilitate interaction between different storage systems
- Pipeline-like syntax for defining synchronization actions on a secret level
- Advanced templating capabilities for transforming secrets

| **Supported store**                                                      | **Status** |
|--------------------------------------------------------------------------|------------|
| [HashiCorp Vault](https://www.vaultproject.io)                           | alpha      |
| [Local Provider]                                                         | alpha      |
| [AWS Secrets Manager](https://aws.amazon.com/secrets-manager)            | _planned_  |
| [Google Secrets Manager](https://cloud.google.com/secret-manager)        | _planned_  |
| [Azure Key Vault](https://azure.microsoft.com/en-us/services/key-vault/) | _planned_  |
| [Kubernetes Secret](https://kubernetes.io/)                              | _planned_  |

Check details about upcoming features by visiting the [project issue](https://github.com/bank-vaults/secret-sync/issues) board.

## Goals

- Provide safe and simple way to work with secrets
- Expose common API for secret management regardless of the store backend
- Give total control of the secret synchronization process

> Consider a situation where Dev teams need access to secrets from different environments.
> Ops teams can provide access to secrets in the form of an isolated environment (e.g. new Vault instance) synced only with secrets Devs require; all in GitOps way.

## Getting started

To get familiarized, check out the collection of different [examples](examples) using this tool.

## Documentation

Check out the [project documentation](docs) or [pkg.go.dev](https://pkg.go.dev/mod/github.com/bank-vaults/secret-sync).

## Development

Install [Go](https://go.dev/dl/) on your computer then run `make deps` to install the rest of the dependencies.

Make sure Docker is installed with Compose and Buildx.

Run project dependencies:

```shell
make up
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

Build artifacts locally:

```shell
make artifacts
```

Once you are done either stop or tear down dependencies:

```shell
make stop

# OR

make down
```

## License

The project is licensed under the [Apache 2.0 License](https://github.com/bank-vaults/secret-sync/blob/master/LICENSE).
