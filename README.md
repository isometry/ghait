# ghait

`ghait` is a reusable Go module and CLI tool designed to simplify generation of ephemeral GitHub App Installation Tokens.
It directly supports multiple Key Management Service (KMS) providers, including AWS, GCP, and Vault, to securely sign requests.

## Features

- Easily generate ephemeral GitHub App Installation Tokens
- Support for multiple KMS providers: File, AWS, GCP, Vault
- Support for restricting repositories and permissions per token
- Fully configurable via environment variables and command-line flags

## Installation

To install the CLI tool, use the following command:

```sh
go install github.com/isometry/ghait/cmd/ghait@latest
```

### Homebrew

```sh
brew install isometry/tap/ghait
```

## Usage

### CLI Interface

The `ghait` CLI tool can be used to generate ephemeral GitHub App Installation Tokens, each valid for 1-hour. Below is a brief description of the available flags:

```shell
Usage:
  ghait [flags]

Flags:
  -a, --app-id int                  App ID (required)
  -i, --installation-id int         Installation ID (required)
  -k, --key string                  Private key or identifier (required)
  -P, --provider string             KMS provider (supported: [file,aws,gcp,vault]) (default "file")
  -r, --repository strings          Repository names to grant access to (default all)
  -p, --permission stringToString   Restricted permissions to grant (default all)
  -h, --help                        help for ghait
  -v, --version                     version for ghait
```

### Example

To generate a GitHub App installation token using the CLI, run:

```sh
export GHAIT_APP_ID=12345
export GHAIT_INSTALLATION_ID=67890
ghait -k private.pem
ghait --key private.pem --repo test-repo --permissions contents=read
ghait --provider aws --key alias/github
ghait --provider vault --key transit/sign/github --repo test-repo --permission contents=read,metadata=read
```

## Providers

Various KMS providers are implemented, each conforming to the `Signer` interface of [`bradleyfalzon/ghinstallation/v2`](https://github.com/bradleyfalzon/ghinstallation).

### File (registered by default)

The `file` provider expects `key` to be the path to a file holding your GitHub App private key, or alternatively the full contents of the key itself. This provider is registered by default when importing the `ghait` library. To disable it, build with `-tags ghait.no_file`.

### AWS

The `aws` provider offloads JWT token signing to AWS KMS. `key` takes the form of a KMS key reference.
Usage relies on standard AWS configuration and credentials being available to the app.

### GCP

The `gcp` provider offloads JWT token signing to GCP KMS. `key` takes the form of a KMS key reference.
Usage relies on standard GCP configuration and credentials being available to the app.

### Vault

The `vault` provider offloads JWT token signing to GCP KMS. `key` takes the form of a transit secrets engine signing path `<mountpoint>/sign/<name>`, for example `transit/sign/github`.
Usage relies on standard Vault configuration and credentials being available to the app.

### Library Usage

When using `ghait` as a library, the `file` provider is registered by default. AWS/GCP KMS and Vault providers can be enabled in two ways:

#### 1. Explicit underscore imports

Add underscore imports in your consuming code to register providers at compile time:

```go
import _ "github.com/isometry/ghait/provider/aws"
import _ "github.com/isometry/ghait/provider/gcp"
import _ "github.com/isometry/ghait/provider/vault"
```

#### 2. Build tags

Enable providers without modifying source code using build tags. This is useful for users of ghait-consuming binaries who want to enable alternate providers:

```sh
go build -tags ghait.aws
go build -tags ghait.gcp
go build -tags ghait.vault
go build -tags ghait.aws,ghait.gcp,ghait.vault
```

#### Disabling providers

Individual providers can be disabled at build time using opt-out build tags, even if they are underscore-imported by the source code:

```sh
go build -tags ghait.no_aws
go build -tags ghait.no_gcp
go build -tags ghait.no_vault
go build -tags ghait.no_file
```

## Environment Variables

You can also configure the CLI using environment variables:

- `GHAIT_APP_ID`: GitHub App ID
- `GHAIT_INSTALLATION_ID`: GitHub App Installation ID
- `GHAIT_KEY`: Private key or identifier
- `GHAIT_PROVIDER`: KMS provider (supported: file, aws, gcp, vault)
- `GHAIT_REPOSITORY`: Repositories to grant access to (space-delimited)
- `GHAIT_PERMISSION`: Restricted permissions to grant (JSON map)
- `GHAIT_VALIDATE_KEY`: Set to any non-empty value to enable key validation at startup (see below)

## Key Validation

Key validation can be enabled to fail fast on misconfigured keys. This check verifies that the key is an enabled RSA 2048-bit key suitable for RS256 signing (the only key type used by GitHub Apps).

Enable via any of:
- Environment variable: `GHAIT_VALIDATE_KEY=1`
- CLI flag: `--validate-key`
- Programmatically: `ghait.NewConfig(...).WithValidateKey(true)`

By default, key checks are **disabled** to minimise the required permissions. Enabling them requires additional read/describe permissions beyond signing:

| Provider | Additional Permission Required |
|----------|-------------------------------|
| AWS      | `kms:DescribeKey`             |
| Azure    | `keys/get`                    |
| GCP      | `cloudkms.cryptoKeyVersions.get` |
| Vault    | `read` on `<mount>/keys/<name>` |
| File     | _(none — validated at construction)_ |

## Programmatic Usage

To use this module programmatically, you can create a new instance of ghait and generate a token as shown below:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/isometry/ghait"
    "github.com/google/go-github/v84/github"

    _ "github.com/isometry/ghait/provider/aws" // register the AWS provider
)

func main() {
    ctx := context.Background()
    config := ghait.NewConfig(12345, 67890, "aws", "alias/github")

    factory, err := ghait.NewGHAIT(ctx, config)
    if err != nil {
        log.Fatalf("failed to create ghait instance: %v", err)
    }

    installationToken, err := factory.NewToken(ctx)
    if err != nil {
        log.Fatalf("failed to create installation token: %v", err)
    }

    fmt.Println(installationToken.GetToken())
}
```

## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

This project is licensed under the Apache License 2.0.
