# terraform-provider-vault-secrets-arrstack

A Terraform/OpenTofu provider that manages the **arrstack dynamic secrets engine**
([`vault-plugin-secrets-arrstack`](https://git.unkin.net/unkin/vault-plugin-secrets-arrstack))
on HashiCorp Vault or OpenBao, so the engine's mount, config, and roles can be
driven declaratively (e.g. from `terraform-vault`). The engine mints short-lived
arrproxy API keys scoped to the arr apps (Sonarr, Radarr, Prowlarr).

Source address: `artifactapi.k8s.syd1.au.unkin.net/terraform-unkin/vault-secrets-arrstack`
(declare it under the local name `arrstack`, so its resources are `arrstack_*`).

## Resources

| Resource | Manages |
|----------|---------|
| `arrstack_secret_backend` | Mounts the engine at a path and writes its `config` (arrproxy base URL, request timeout, seeded admin token, optional CA cert). |
| `arrstack_secret_backend_role` | A role: `apps` (subset of sonarr/radarr/prowlarr), `ttl`, `max_ttl`. |

## Usage

```hcl
terraform {
  required_providers {
    arrstack = {
      source  = "artifactapi.k8s.syd1.au.unkin.net/terraform-unkin/vault-secrets-arrstack"
      version = "0.1.0"
    }
  }
}

provider "arrstack" {
  # address / token fall back to VAULT_ADDR / VAULT_TOKEN.
}

resource "arrstack_secret_backend" "arrstack" {
  path        = "arrstack"
  base_url    = "https://arrstack.unkin.net"
  admin_token = var.arrproxy_admin_token
}

resource "arrstack_secret_backend_role" "all" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "all"
  apps    = ["sonarr", "radarr", "prowlarr"]
  ttl     = 60
  max_ttl = 86400
}
```

### Notes

- `admin_token` is write-only: Vault never returns it, so it is preserved in
  Terraform state and does not show drift. `ca_cert` is likewise write-only and
  preserved; omit it to use the system trust store.
- Writing `config` makes the engine authenticate against arrproxy as an admin,
  so a bad URL or token fails the apply.
- `apps` is required; entries must be a subset of `sonarr`, `radarr`, `prowlarr`.

## Import

```sh
terraform import arrstack_secret_backend.arrstack arrstack
terraform import arrstack_secret_backend_role.all arrstack/roles/all
```

## Development

```sh
make build      # build the provider binary
make install    # install into ~/.terraform.d/plugins for local use
make test       # unit tests (race)
make package    # build the release zip
```

Releases are tag-driven (`make patch|minor|major`): a Woodpecker pipeline builds
`terraform-provider-vault-secrets-arrstack_<version>_linux_amd64.zip` and PUTs it
to the artifactapi terraform registry
(`.../api/v2/remotes/terraform-unkin/files/terraform-unkin/vault-secrets-arrstack/<file>`),
which signs it server-side. Install it via the bare `source` address above.
