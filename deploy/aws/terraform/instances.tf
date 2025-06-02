# VPCs for each region
resource "aws_vpc" "chuckd_vpc_us_east_1" {
  provider             = aws.us_east_1
  cidr_block           = "10.0.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "chuckd-vpc-us-east-1"
  }
}

resource "aws_vpc" "chuckd_vpc_us_west_1" {
  provider             = aws.us_west_1
  cidr_block           = "10.1.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "chuckd-vpc-us-west-1"
  }
}

resource "aws_vpc" "chuckd_vpc_ap_northeast_1" {
  provider             = aws.ap_northeast_1
  cidr_block           = "10.2.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "chuckd-vpc-ap-northeast-1"
  }
}

resource "aws_vpc" "chuckd_vpc_eu_central_1" {
  provider             = aws.eu_central_1
  cidr_block           = "10.3.0.0/16"
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = {
    Name = "chuckd-vpc-eu-central-1"
  }
}

# VPC Peering connections
resource "aws_vpc_peering_connection" "chuckd_peer_us_east_1_us_west_1" {
  provider    = aws.us_east_1
  vpc_id      = aws_vpc.chuckd_vpc_us_east_1.id
  peer_vpc_id = aws_vpc.chuckd_vpc_us_west_1.id
  peer_region = "us-west-1"
  auto_accept = false

  tags = {
    Name = "chuckd-peer-us-east-1-to-us-west-1"
  }
}

resource "aws_vpc_peering_connection" "chuckd_peer_us_east_1_ap_northeast_1" {
  provider    = aws.us_east_1
  vpc_id      = aws_vpc.chuckd_vpc_us_east_1.id
  peer_vpc_id = aws_vpc.chuckd_vpc_ap_northeast_1.id
  peer_region = "ap-northeast-1"
  auto_accept = false

  tags = {
    Name = "chuckd-peer-us-east-1-to-ap-northeast-1"
  }
}

resource "aws_vpc_peering_connection" "chuckd_peer_us_east_1_eu_central_1" {
  provider    = aws.us_east_1
  vpc_id      = aws_vpc.chuckd_vpc_us_east_1.id
  peer_vpc_id = aws_vpc.chuckd_vpc_eu_central_1.id
  peer_region = "eu-central-1"
  auto_accept = false

  tags = {
    Name = "chuckd-peer-us-east-1-to-eu-central-1"
  }
}

resource "aws_vpc_peering_connection" "chuckd_peer_us_west_1_ap_northeast_1" {
  provider    = aws.us_west_1
  vpc_id      = aws_vpc.chuckd_vpc_us_west_1.id
  peer_vpc_id = aws_vpc.chuckd_vpc_ap_northeast_1.id
  peer_region = "ap-northeast-1"
  auto_accept = false

  tags = {
    Name = "chuckd-peer-us-west-1-to-ap-northeast-1"
  }
}

resource "aws_vpc_peering_connection" "chuckd_peer_us_west_1_eu_central_1" {
  provider    = aws.us_west_1
  vpc_id      = aws_vpc.chuckd_vpc_us_west_1.id
  peer_vpc_id = aws_vpc.chuckd_vpc_eu_central_1.id
  peer_region = "eu-central-1"
  auto_accept = false

  tags = {
    Name = "chuckd-peer-us-west-1-to-eu-central-1"
  }
}

resource "aws_vpc_peering_connection" "chuckd_peer_ap_northeast_1_eu_central_1" {
  provider    = aws.ap_northeast_1
  vpc_id      = aws_vpc.chuckd_vpc_ap_northeast_1.id
  peer_vpc_id = aws_vpc.chuckd_vpc_eu_central_1.id
  peer_region = "eu-central-1"
  auto_accept = false

  tags = {
    Name = "chuckd-peer-ap-northeast-1-to-eu-central-1"
  }
}

# VPC Peering Accepters
resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept_us_east_1_us_west_1" {
  provider                  = aws.us_west_1
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_us_west_1.id
  auto_accept               = true
}

resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept_us_east_1_ap_northeast_1" {
  provider                  = aws.ap_northeast_1
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_ap_northeast_1.id
  auto_accept               = true
}

resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept_us_east_1_eu_central_1" {
  provider                  = aws.eu_central_1
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_eu_central_1.id
  auto_accept               = true
}

resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept_us_west_1_ap_northeast_1" {
  provider                  = aws.ap_northeast_1
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_west_1_ap_northeast_1.id
  auto_accept               = true
}

resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept_us_west_1_eu_central_1" {
  provider                  = aws.eu_central_1
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_west_1_eu_central_1.id
  auto_accept               = true
}

resource "aws_vpc_peering_connection_accepter" "chuckd_peer_accept_ap_northeast_1_eu_central_1" {
  provider                  = aws.eu_central_1
  vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_ap_northeast_1_eu_central_1.id
  auto_accept               = true
}

# Internet Gateways
resource "aws_internet_gateway" "chuckd_igw_us_east_1" {
  provider = aws.us_east_1
  vpc_id   = aws_vpc.chuckd_vpc_us_east_1.id

  tags = {
    Name = "chuckd-igw-us-east-1"
  }
}

resource "aws_internet_gateway" "chuckd_igw_us_west_1" {
  provider = aws.us_west_1
  vpc_id   = aws_vpc.chuckd_vpc_us_west_1.id

  tags = {
    Name = "chuckd-igw-us-west-1"
  }
}

resource "aws_internet_gateway" "chuckd_igw_ap_northeast_1" {
  provider = aws.ap_northeast_1
  vpc_id   = aws_vpc.chuckd_vpc_ap_northeast_1.id

  tags = {
    Name = "chuckd-igw-ap-northeast-1"
  }
}

resource "aws_internet_gateway" "chuckd_igw_eu_central_1" {
  provider = aws.eu_central_1
  vpc_id   = aws_vpc.chuckd_vpc_eu_central_1.id

  tags = {
    Name = "chuckd-igw-eu-central-1"
  }
}

# Subnets
resource "aws_subnet" "chuckd_subnet_us_east_1" {
  provider          = aws.us_east_1
  vpc_id            = aws_vpc.chuckd_vpc_us_east_1.id
  cidr_block        = "10.0.1.0/24"
  availability_zone = "us-east-1a"

  map_public_ip_on_launch = true

  tags = {
    Name = "chuckd-subnet-us-east-1"
  }
}

resource "aws_subnet" "chuckd_subnet_us_west_1" {
  provider          = aws.us_west_1
  vpc_id            = aws_vpc.chuckd_vpc_us_west_1.id
  cidr_block        = "10.1.1.0/24"
  availability_zone = "us-west-1a"

  map_public_ip_on_launch = true

  tags = {
    Name = "chuckd-subnet-us-west-1"
  }
}

resource "aws_subnet" "chuckd_subnet_ap_northeast_1" {
  provider          = aws.ap_northeast_1
  vpc_id            = aws_vpc.chuckd_vpc_ap_northeast_1.id
  cidr_block        = "10.2.1.0/24"
  availability_zone = "ap-northeast-1a"

  map_public_ip_on_launch = true

  tags = {
    Name = "chuckd-subnet-ap-northeast-1"
  }
}

resource "aws_subnet" "chuckd_subnet_eu_central_1" {
  provider          = aws.eu_central_1
  vpc_id            = aws_vpc.chuckd_vpc_eu_central_1.id
  cidr_block        = "10.3.1.0/24"
  availability_zone = "eu-central-1a"

  map_public_ip_on_launch = true

  tags = {
    Name = "chuckd-subnet-eu-central-1"
  }
}

# Route Tables
resource "aws_route_table" "chuckd_rt_us_east_1" {
  provider = aws.us_east_1
  vpc_id   = aws_vpc.chuckd_vpc_us_east_1.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.chuckd_igw_us_east_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_us_west_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_us_west_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_ap_northeast_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_eu_central_1.id
  }

  tags = {
    Name = "chuckd-rt-us-east-1"
  }
}

resource "aws_route_table" "chuckd_rt_us_west_1" {
  provider = aws.us_west_1
  vpc_id   = aws_vpc.chuckd_vpc_us_west_1.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.chuckd_igw_us_west_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_us_east_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_us_west_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_west_1_ap_northeast_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_west_1_eu_central_1.id
  }

  tags = {
    Name = "chuckd-rt-us-west-1"
  }
}

resource "aws_route_table" "chuckd_rt_ap_northeast_1" {
  provider = aws.ap_northeast_1
  vpc_id   = aws_vpc.chuckd_vpc_ap_northeast_1.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.chuckd_igw_ap_northeast_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_us_east_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_ap_northeast_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_us_west_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_west_1_ap_northeast_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_ap_northeast_1_eu_central_1.id
  }

  tags = {
    Name = "chuckd-rt-ap-northeast-1"
  }
}

resource "aws_route_table" "chuckd_rt_eu_central_1" {
  provider = aws.eu_central_1
  vpc_id   = aws_vpc.chuckd_vpc_eu_central_1.id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.chuckd_igw_eu_central_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_us_east_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_east_1_eu_central_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_us_west_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_us_west_1_eu_central_1.id
  }

  route {
    cidr_block                = aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block
    vpc_peering_connection_id = aws_vpc_peering_connection.chuckd_peer_ap_northeast_1_eu_central_1.id
  }

  tags = {
    Name = "chuckd-rt-eu-central-1"
  }
}

# Route Table Associations
resource "aws_route_table_association" "chuckd_rta_us_east_1" {
  provider       = aws.us_east_1
  subnet_id      = aws_subnet.chuckd_subnet_us_east_1.id
  route_table_id = aws_route_table.chuckd_rt_us_east_1.id
}

resource "aws_route_table_association" "chuckd_rta_us_west_1" {
  provider       = aws.us_west_1
  subnet_id      = aws_subnet.chuckd_subnet_us_west_1.id
  route_table_id = aws_route_table.chuckd_rt_us_west_1.id
}

resource "aws_route_table_association" "chuckd_rta_ap_northeast_1" {
  provider       = aws.ap_northeast_1
  subnet_id      = aws_subnet.chuckd_subnet_ap_northeast_1.id
  route_table_id = aws_route_table.chuckd_rt_ap_northeast_1.id
}

resource "aws_route_table_association" "chuckd_rta_eu_central_1" {
  provider       = aws.eu_central_1
  subnet_id      = aws_subnet.chuckd_subnet_eu_central_1.id
  route_table_id = aws_route_table.chuckd_rt_eu_central_1.id
}

# Security Groups
resource "aws_security_group" "chuckd_sg_us_east_1" {
  provider    = aws.us_east_1
  name        = "chuckd-sg-us-east-1"
  description = "Security group for chuckd server in us-east-1"
  vpc_id      = aws_vpc.chuckd_vpc_us_east_1.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = var.app_port
    to_port   = var.app_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  ingress {
    from_port = var.metrics_port
    to_port   = var.metrics_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "chuckd-sg-us-east-1"
  }
}

resource "aws_security_group" "chuckd_sg_us_west_1" {
  provider    = aws.us_west_1
  name        = "chuckd-sg-us-west-1"
  description = "Security group for chuckd server in us-west-1"
  vpc_id      = aws_vpc.chuckd_vpc_us_west_1.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = var.app_port
    to_port   = var.app_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  ingress {
    from_port = var.metrics_port
    to_port   = var.metrics_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "chuckd-sg-us-west-1"
  }
}

resource "aws_security_group" "chuckd_sg_ap_northeast_1" {
  provider    = aws.ap_northeast_1
  name        = "chuckd-sg-ap-northeast-1"
  description = "Security group for chuckd server in ap-northeast-1"
  vpc_id      = aws_vpc.chuckd_vpc_ap_northeast_1.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = var.app_port
    to_port   = var.app_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  ingress {
    from_port = var.metrics_port
    to_port   = var.metrics_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "chuckd-sg-ap-northeast-1"
  }
}

resource "aws_security_group" "chuckd_sg_eu_central_1" {
  provider    = aws.eu_central_1
  name        = "chuckd-sg-eu-central-1"
  description = "Security group for chuckd server in eu-central-1"
  vpc_id      = aws_vpc.chuckd_vpc_eu_central_1.id

  # SSH access
  ingress {
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  ingress {
    from_port = var.app_port
    to_port   = var.app_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  ingress {
    from_port = var.metrics_port
    to_port   = var.metrics_port
    protocol  = "tcp"
    cidr_blocks = [
      "0.0.0.0/0",
      aws_vpc.chuckd_vpc_us_east_1.cidr_block,
      aws_vpc.chuckd_vpc_us_west_1.cidr_block,
      aws_vpc.chuckd_vpc_ap_northeast_1.cidr_block,
      aws_vpc.chuckd_vpc_eu_central_1.cidr_block
    ]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = {
    Name = "chuckd-sg-eu-central-1"
  }
}

# Route53 Zone
resource "aws_route53_zone" "chuckd" {
  provider = aws.us_east_1
  name     = "chuckd.internal"

  vpc {
    vpc_id = aws_vpc.chuckd_vpc_us_east_1.id
  }
}

# Route53 Zone Associations
resource "aws_route53_zone_association" "secondary_us_west_1" {
  provider = aws.us_west_1
  zone_id  = aws_route53_zone.chuckd.id
  vpc_id   = aws_vpc.chuckd_vpc_us_west_1.id
}

resource "aws_route53_zone_association" "secondary_ap_northeast_1" {
  provider = aws.ap_northeast_1
  zone_id  = aws_route53_zone.chuckd.id
  vpc_id   = aws_vpc.chuckd_vpc_ap_northeast_1.id
}

resource "aws_route53_zone_association" "secondary_eu_central_1" {
  provider = aws.eu_central_1
  zone_id  = aws_route53_zone.chuckd.id
  vpc_id   = aws_vpc.chuckd_vpc_eu_central_1.id
}

# EC2 Instances
resource "aws_instance" "chuckd_server_us_east_1" {
  provider      = aws.us_east_1
  ami           = var.ami_ids["us-east-1"]
  instance_type = var.instance_type

  subnet_id                   = aws_subnet.chuckd_subnet_us_east_1.id
  vpc_security_group_ids      = [aws_security_group.chuckd_sg_us_east_1.id]
  associate_public_ip_address = true

  user_data = <<-EOF
              #!/bin/bash
              mkdir -p /etc/chuckd
              echo "${file(var.config_files["us-east-1"])}" > /etc/chuckd/config.json

              cat > /etc/systemd/system/chuckd.service <<'EOL'
              [Unit]
              Description=Chuckd Server
              After=network.target

              [Service]
              Type=simple
              ExecStart=/usr/local/bin/chuckd ${lookup(var.chuckd_args, "us-east-1", "")}
              Restart=no

              [Install]
              WantedBy=multi-user.target
              EOL

              systemctl daemon-reload
              systemctl enable chuckd
              systemctl start chuckd
              EOF

  tags = {
    Name = "chuckd-server-us-east-1"
  }
}

resource "aws_instance" "chuckd_server_us_west_1" {
  provider      = aws.us_west_1
  ami           = var.ami_ids["us-west-1"]
  instance_type = var.instance_type

  subnet_id                   = aws_subnet.chuckd_subnet_us_west_1.id
  vpc_security_group_ids      = [aws_security_group.chuckd_sg_us_west_1.id]
  associate_public_ip_address = true

  user_data = <<-EOF
              #!/bin/bash
              mkdir -p /etc/chuckd
              echo "${file(var.config_files["us-west-1"])}" > /etc/chuckd/config.json

              cat > /etc/systemd/system/chuckd.service <<'EOL'
              [Unit]
              Description=Chuckd Server
              After=network.target

              [Service]
              Type=simple
              ExecStart=/usr/local/bin/chuckd ${lookup(var.chuckd_args, "us-west-1", "")}
              Restart=no

              [Install]
              WantedBy=multi-user.target
              EOL

              systemctl daemon-reload
              systemctl enable chuckd
              systemctl start chuckd
              EOF

  tags = {
    Name = "chuckd-server-us-west-1"
  }
}

resource "aws_instance" "chuckd_server_ap_northeast_1" {
  provider      = aws.ap_northeast_1
  ami           = var.ami_ids["ap-northeast-1"]
  instance_type = var.instance_type

  subnet_id                   = aws_subnet.chuckd_subnet_ap_northeast_1.id
  vpc_security_group_ids      = [aws_security_group.chuckd_sg_ap_northeast_1.id]
  associate_public_ip_address = true

  user_data = <<-EOF
              #!/bin/bash
              mkdir -p /etc/chuckd
              echo "${file(var.config_files["ap-northeast-1"])}" > /etc/chuckd/config.json

              cat > /etc/systemd/system/chuckd.service <<'EOL'
              [Unit]
              Description=Chuckd Server
              After=network.target

              [Service]
              Type=simple
              ExecStart=/usr/local/bin/chuckd ${lookup(var.chuckd_args, "ap-northeast-1", "")}
              Restart=no

              [Install]
              WantedBy=multi-user.target
              EOL

              systemctl daemon-reload
              systemctl enable chuckd
              systemctl start chuckd
              EOF

  tags = {
    Name = "chuckd-server-ap-northeast-1"
  }
}

resource "aws_instance" "chuckd_server_eu_central_1" {
  provider      = aws.eu_central_1
  ami           = var.ami_ids["eu-central-1"]
  instance_type = var.instance_type

  subnet_id                   = aws_subnet.chuckd_subnet_eu_central_1.id
  vpc_security_group_ids      = [aws_security_group.chuckd_sg_eu_central_1.id]
  associate_public_ip_address = true

  user_data = <<-EOF
              #!/bin/bash
              mkdir -p /etc/chuckd
              echo "${file(var.config_files["eu-central-1"])}" > /etc/chuckd/config.json

              cat > /etc/systemd/system/chuckd.service <<'EOL'
              [Unit]
              Description=Chuckd Server
              After=network.target

              [Service]
              Type=simple
              ExecStart=/usr/local/bin/chuckd ${lookup(var.chuckd_args, "eu-central-1", "")}
              Restart=no

              [Install]
              WantedBy=multi-user.target
              EOL

              systemctl daemon-reload
              systemctl enable chuckd
              systemctl start chuckd
              EOF

  tags = {
    Name = "chuckd-server-eu-central-1"
  }
}

# Route53 Records
resource "aws_route53_record" "chuckd_dns_us_east_1" {
  provider = aws.us_east_1
  zone_id  = aws_route53_zone.chuckd.id
  name     = "chuckd-us-east-1.chuckd.internal"
  type     = "A"
  ttl      = "300"
  records  = [aws_instance.chuckd_server_us_east_1.private_ip]
}

resource "aws_route53_record" "chuckd_dns_us_west_1" {
  provider = aws.us_west_1
  zone_id  = aws_route53_zone.chuckd.id
  name     = "chuckd-us-west-1.chuckd.internal"
  type     = "A"
  ttl      = "300"
  records  = [aws_instance.chuckd_server_us_west_1.private_ip]
}

resource "aws_route53_record" "chuckd_dns_ap_northeast_1" {
  provider = aws.ap_northeast_1
  zone_id  = aws_route53_zone.chuckd.id
  name     = "chuckd-ap-northeast-1.chuckd.internal"
  type     = "A"
  ttl      = "300"
  records  = [aws_instance.chuckd_server_ap_northeast_1.private_ip]
}

resource "aws_route53_record" "chuckd_dns_eu_central_1" {
  provider = aws.eu_central_1
  zone_id  = aws_route53_zone.chuckd.id
  name     = "chuckd-eu-central-1.chuckd.internal"
  type     = "A"
  ttl      = "300"
  records  = [aws_instance.chuckd_server_eu_central_1.private_ip]
} 