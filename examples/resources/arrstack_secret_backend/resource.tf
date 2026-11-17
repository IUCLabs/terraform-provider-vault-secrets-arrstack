variable "arrproxy_admin_token" {
  type      = string
  sensitive = true
}

resource "arrstack_secret_backend" "arrstack" {
  path        = "arrstack"
  base_url    = "https://arrproxy.example.com"
  admin_token = var.arrproxy_admin_token

  request_timeout_seconds = 30
}
