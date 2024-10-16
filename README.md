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

### File

The `file` provider expects `key` to be the path to a file holding your GitHub App private key, or alternatively the full contents of the key itself.

Disable inclusion with the `no_file` build tag.

### AWS

The `aws` provider offloads JWT token signing to AWS KMS. `key` takes the form of a KMS key reference.
Usage relies on standard AWS configuration and credentials being available to the app.

Disable inclusion with the `no_aws` build tag.

### GCP

The `gcp` provider offloads JWT token signing to GCP KMS. `key` takes the form of a KMS key reference.
Usage relies on standard GCP configuration and credentials being available to the app.

Disable inclusion with the `no_gcp` build tag.

### Vault

The `vault` provider offloads JWT token signing to GCP KMS. `key` takes the form of a transit secrets engine signing path `<mountpoint>/sign/<name>`, for example `transit/sign/github`.
Usage relies on standard Vault configuration and credentials being available to the app.

Disable inclusion with the `no_vault` build tag.

## Environment Variables

You can also configure the CLI using environment variables:

- `GHAIT_APP_ID`: GitHub App ID
- `GHAIT_INSTALLATION_ID`: GitHub App Installation ID
- `GHAIT_KEY`: Private key or identifier
- `GHAIT_PROVIDER`: KMS provider (supported: file, aws, gcp, vault)
- `GHAIT_REPOSITORY`: Repositories to grant access to (space-delimited)
- `GHAIT_PERMISSION`: Restricted permissions to grant (JSON map)

## Programmatic Usage

To use this module programmatically, you can create a new instance of ghait and generate a token as shown below:

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/isometry/ghait"
    "github.com/google/go-github/v66/github"
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
