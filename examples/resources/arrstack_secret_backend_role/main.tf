# A role that mints short-lived arrproxy API keys scoped to a subset of the arr
# apps. Reading arrstack/creds/<role> returns a lease-bound key that the engine
# revokes when the lease ends.
resource "arrstack_secret_backend_role" "sonarr" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "sonarr"
  apps    = ["sonarr"]

  ttl     = 60    # 1m
  max_ttl = 86400 # 24h
}

# The same role narrowed to read-only traffic: keys minted from it may only
# issue GET/HEAD against Sonarr. Omitting methods leaves a key unrestricted.
resource "arrstack_secret_backend_role" "sonarr_ro" {
  backend = arrstack_secret_backend.arrstack.path
  name    = "sonarr-ro"
  apps    = ["sonarr"]
  methods = ["GET", "HEAD"]

  ttl     = 60    # 1m
  max_ttl = 86400 # 24h
}
