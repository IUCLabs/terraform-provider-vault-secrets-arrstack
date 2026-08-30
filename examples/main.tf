terraform {
  required_providers {
    arrstack = {
      source  = "artifactapi.k8s.syd1.au.unkin.net/terraform-unkin/vault-secrets-arrstack"
      version = "0.1.0"
    }
  }
}

provider "arrstack" {
  # address defaults to $VAULT_ADDR, token to $VAULT_TOKEN
}

variable "arrproxy_admin_token" {
  type      = string
  sensitive = true
}

resource "arrstack_secret_backend" "arrstack" {
  path        = "arrstack"
  base_url    = "https://arrstack.unkin.net"
  admin_token = var.arrproxy_admin_token
}

# Role that mints an arrproxy API key scoped to all three arr apps.
resource "arrstack_secret_backend_role" "all" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "all"
  apps    = ["sonarr", "radarr", "prowlarr"]
  ttl     = 60
  max_ttl = 86400
}

# Role scoped to Prowlarr only.
resource "arrstack_secret_backend_role" "prowlarr" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "prowlarr"
  apps    = ["prowlarr"]
  ttl     = 60
  max_ttl = 86400
}

# Role scoped to read-only traffic across all three apps.
resource "arrstack_secret_backend_role" "readonly" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "readonly"
  apps    = ["sonarr", "radarr", "prowlarr"]
  methods = ["GET", "HEAD"]
  ttl     = 60
  max_ttl = 86400
}
