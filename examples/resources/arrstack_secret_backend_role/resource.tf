# Tokens minted from arrstack/creds/sonarr can reach Sonarr only.
resource "arrstack_secret_backend_role" "sonarr" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "sonarr"
  apps    = ["sonarr"]
  ttl     = 3600
  max_ttl = 86400
}

# Read-only: tokens may only send GET and HEAD. Omit methods for unrestricted.
resource "arrstack_secret_backend_role" "sonarr_ro" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "sonarr-ro"
  apps    = ["sonarr"]
  methods = ["GET", "HEAD"]
  ttl     = 3600
  max_ttl = 86400
}
