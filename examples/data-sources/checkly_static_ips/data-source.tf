# All static IP addresses that Checkly's runners use.
data "checkly_static_ips" "all" {}

# Only the IPv4 addresses of two locations.
data "checkly_static_ips" "eu_ipv4" {
  locations = ["eu-central-1", "eu-west-1"]
  ip_family = "IPv4"
}

# Allow traffic from those runners to a service behind an AWS security group.
# The addresses are already in CIDR notation, such as "3.64.0.1/32".
resource "aws_security_group_rule" "checkly_ingress" {
  type              = "ingress"
  from_port         = 443
  to_port           = 443
  protocol          = "tcp"
  cidr_blocks       = tolist(data.checkly_static_ips.eu_ipv4.addresses)
  security_group_id = "sg-0123456789abcdef0"
}
