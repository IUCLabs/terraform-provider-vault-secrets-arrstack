# Terraform Provider for the arrstack Vault secrets engine

This provider manages the arrstack secrets engine
([vault-plugin-secrets-arrstack](https://github.com/IUCLabs/vault-plugin-secrets-arrstack))
on HashiCorp Vault or OpenBao: the engine's mount, its arrproxy connection
config, and its roles. The engine mints short-lived arrproxy machine tokens
scoped to Sonarr, Radarr, and Prowlarr.

Works with Terraform and OpenTofu.

- [Provider documentation](docs/index.md)
- [`arrstack_secret_backend`](docs/resources/secret_backend.md)
- [`arrstack_secret_backend_role`](docs/resources/secret_backend_role.md)

## Requirements

- Terraform 1.0+ or OpenTofu 1.6+
- Vault 1.12+ or OpenBao 2.x with
  [vault-plugin-secrets-arrstack](https://github.com/IUCLabs/vault-plugin-secrets-arrstack)
  registered in the plugin catalog (0.2.0+ for `methods`)
- Go 1.25+ to build from source

## Usage

Prebuilt provider archives are on the [releases page](https://github.com/IUCLabs/terraform-provider-vault-secrets-arrstack/releases).

The registry type is `vault-secrets-arrstack`; declare it under the local name
`arrstack` so its resources are `arrstack_*`.

```hcl
terraform {
  required_providers {
    arrstack = {
      source = "IUCLabs/vault-secrets-arrstack"
    }
  }
}

# address and token fall back to VAULT_ADDR and VAULT_TOKEN.
provider "arrstack" {}

resource "arrstack_secret_backend" "arrstack" {
  path        = "arrstack"
  base_url    = "https://arrproxy.example.com"
  admin_token = var.arrproxy_admin_token
}

resource "arrstack_secret_backend_role" "sonarr_ro" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "sonarr-ro"
  apps    = ["sonarr"]
  methods = ["GET", "HEAD"]
  ttl     = 3600
  max_ttl = 86400
}
```

Tokens are then minted with `vault read arrstack/creds/sonarr-ro`.

## Development

```sh
make build   # build the provider binary
make test    # unit tests
make lint    # go vet
make docs    # regenerate docs/ with tfplugindocs
```

`make install` copies the binary into `~/.terraform.d/plugins` for local
testing.

## License

[MIT](LICENSE)
