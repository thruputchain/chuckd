# Create a VPC for each region
resource "aws_vpc" "chuckd_vpc" {
  for_each = toset(var.regions)

  provider             = aws[replace(each.key, "-", "_")]
  cidr_block           = "10.${index(var.regions, each.key)}.0.0/16" # Unique CIDR for each region
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "chuckd-vpc-${each.key}"
  }
}

# Create VPC peering connections between all regions
resource "aws_vpc_peering_connection" "chuckd_peer" {
  for_each = {
    for pair in [
      for pair in setproduct(var.regions, var.regions) : pair
      if pair[0] < pair[1]
    ] : "${pair[0]}-${pair[1]}" => pair
  }

  provider    = aws[replace(each.value[0], "-", "_")]
  vpc_id      = aws_vpc.chuckd_vpc[each.value[0]].id
  peer_vpc_id = aws_vpc.chuckd_vpc[each.value[1]].id
  peer_region = each.value[1]
  auto_accept = false

  tags = {
    Name = "chuckd-peer-${each.value[0]}-to-${each.value[1]}"
  }
}

# Accept VPC peering connections in peer regions
resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept" {
  for_each = {
    for pair in [
      for pair in setproduct(var.regions, var.regions) : pair
      if pair[0] < pair[1]
    ] : "${pair[0]}-${pair[1]}" => pair
  }

  provider                  = aws[replace(each.value[1], "-", "_")]
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer[each.key].id
  auto_accept               = true
}

# Create an Internet Gateway for each VPC
resource "aws_internet_gateway" "chuckd_igw" {
  for_each = toset(var.regions)

  provider = aws[replace(each.key, "-", "_")]
  vpc_id   = aws_vpc.chuckd_vpc[each.key].id

  tags = {
    Name = "chuckd-igw-${each.key}"
  }
}

# Create a subnet for each VPC
resource "aws_subnet" "chuckd_subnet" {
  for_each = toset(var.regions)

  provider          = aws[replace(each.key, "-", "_")]
  vpc_id            = aws_vpc.chuckd_vpc[each.key].id
  cidr_block        = "10.${index(var.regions, each.key)}.1.0/24"
  availability_zone = "${each.key}a"

  map_public_ip_on_launch = true

  tags = {
    Name = "chuckd-subnet-${each.key}"
  }
}

# Create a route table for each VPC
resource "aws_route_table" "chuckd_rt" {
  for_each = toset(var.regions)

  provider = aws[replace(each.key, "-", "_")]
  vpc_id   = aws_vpc.chuckd_vpc[each.key].id

  # Route to internet gateway
  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.chuckd_igw[each.key].id
  }

  # Routes to other VPCs via peering connections
  dynamic "route" {
    for_each = toset(var.regions) - toset([each.key])
    content {
      cidr_block                = aws_vpc.chuckd_vpc[route.value].cidr_block
      vpc_peering_connection_id = route.value < each.key ? aws_vpc_peering_connection.chuckd_peer["${route.value}-${each.key}"].id : aws_vpc_peering_connection.chuckd_peer["${each.key}-${route.value}"].id
    }
  }

  tags = {
    Name = "chuckd-rt-${each.key}"
  }
}

# Associate route tables with subnets
resource "aws_route_table_association" "chuckd_rta" {
  for_each = toset(var.regions)

  provider       = aws[replace(each.key, "-", "_")]
  subnet_id      = aws_subnet.chuckd_subnet[each.key].id
  route_table_id = aws_route_table.chuckd_rt[each.key].id
}

# Create security group for each region
resource "aws_security_group" "chuckd_sg" {
  for_each = toset(var.regions)

  provider    = aws[replace(each.key, "-", "_")]
  name        = "chuckd-sg-${each.key}"
  description = "Security group for chuckd server in ${each.key}"
  vpc_id      = aws_vpc.chuckd_vpc[each.key].id

  # Application port - allow from all VPCs and public internet
  ingress {
    from_port = var.app_port
    to_port   = var.app_port
    protocol  = "tcp"
    cidr_blocks = concat(
      ["0.0.0.0/0"],
      [for region in var.regions : aws_vpc.chuckd_vpc[region].cidr_block]
    )
  }

  # Metrics port - allow from all VPCs and public internet
  ingress {
    from_port = var.metrics_port
    to_port   = var.metrics_port
    protocol  = "tcp"
    cidr_blocks = concat(
      ["0.0.0.0/0"],
      [for region in var.regions : aws_vpc.chuckd_vpc[region].cidr_block]
    )
  }

  # Allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "chuckd-sg-${each.key}"
  }
}

# Create Route53 private hosted zone
resource "aws_route53_zone" "chuckd" {
  provider = aws[replace(var.regions[0], "-", "_")]
  name     = "chuckd.internal"

  vpc {
    vpc_id = aws_vpc.chuckd_vpc[var.regions[0]].id
  }
}

# Associate Route53 zone with other VPCs
resource "aws_route53_zone_association" "secondary" {
  for_each = toset([for region in var.regions : region if region != var.regions[0]])

  provider = aws[replace(each.key, "-", "_")]
  zone_id  = aws_route53_zone.chuckd.id
  vpc_id   = aws_vpc.chuckd_vpc[each.key].id
}

# Create EC2 instances
resource "aws_instance" "chuckd_server" {
  for_each = toset(var.regions)

  provider      = aws[replace(each.key, "-", "_")]
  ami           = var.ami_ids[each.key]
  instance_type = var.instance_type

  subnet_id                   = aws_subnet.chuckd_subnet[each.key].id
  vpc_security_group_ids      = [aws_security_group.chuckd_sg[each.key].id]
  associate_public_ip_address = true

  user_data = <<-EOF
              #!/bin/bash
              mkdir -p /etc/chuckd
              echo "${file(var.config_files[each.key])}" > /etc/chuckd/config.json

              # Create systemd service file
              cat > /etc/systemd/system/chuckd.service <<'EOL'
              [Unit]
              Description=Chuckd Server
              After=network.target

              [Service]
              Type=simple
              ExecStart=/usr/local/bin/chuckd --config /etc/chuckd/config.json ${lookup(var.chuckd_args, each.key, "")}
              Restart=no

              [Install]
              WantedBy=multi-user.target
              EOL

              # Reload systemd and start service
              systemctl daemon-reload
              systemctl enable chuckd
              systemctl start chuckd
              EOF

  tags = {
    Name = "chuckd-server-${each.key}"
  }
}

# Create Route53 A records using private IPs
resource "aws_route53_record" "chuckd_dns" {
  for_each = toset(var.regions)

  provider = aws[replace(each.key, "-", "_")]
  zone_id  = aws_route53_zone.chuckd.id
  name     = "chuckd-${each.key}.chuckd.internal"
  type     = "A"
  ttl      = "300"
  records  = [aws_instance.chuckd_server[each.key].private_ip] # Using private IP instead of public IP
} 