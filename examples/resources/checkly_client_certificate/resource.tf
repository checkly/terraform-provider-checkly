variable "acme_client_certificate_passphrase" {
  type      = string
  sensitive = true
}

resource "checkly_client_certificate" "test" {
  host        = "*.acme.com"
  certificate = file("${path.module}/cert.pem")
  private_key = file("${path.module}/key.pem")
  trusted_ca  = file("${path.module}/ca.pem")
  passphrase  = var.acme_client_certificate_passphrase
}
