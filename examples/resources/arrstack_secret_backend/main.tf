terraform {
  required_providers {
    arrstack = {
      source = "artifactapi.k8s.syd1.au.unkin.net/terraform-unkin/vault-secrets-arrstack"
    }
  }
}

provider "arrstack" {
  # address / token fall back to VAULT_ADDR / VAULT_TOKEN.
}

# Mount the arrstack secrets engine and seed it with the arrproxy admin token.
resource "arrstack_secret_backend" "arrstack" {
  path     = "arrstack"
  base_url = "https://arrstack.unkin.net"

  admin_token = var.arrproxy_admin_token # e.g. sourced from vault_kv_secret_v2

  request_timeout_seconds = 30
}

variable "arrproxy_admin_token" {
  type      = string
  sensitive = true
}
