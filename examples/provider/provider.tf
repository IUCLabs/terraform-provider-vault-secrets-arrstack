terraform {
  required_providers {
    arrstack = {
      source = "IUCLabs/vault-secrets-arrstack"
    }
  }
}

# address and token fall back to VAULT_ADDR and VAULT_TOKEN.
provider "arrstack" {
  address = "https://vault.example.com:8200"
}
