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
