packer {
  required_plugins {
    amazon = {
      version = ">= 1.2.8"
      source  = "github.com/hashicorp/amazon"
    }
  }
}

# Variables
variable "aws_build_region" {
  type    = string
  default = "us-east-1"
}

variable "ami_regions" {
  type        = list(string)
  default     = ["us-east-1", "us-west-2"] # Default regions, can be overridden
  description = "List of regions where AMIs should be copied"
}

variable "instance_type" {
  type    = string
  default = "t3.medium"
}

variable "github_token" {
  type        = string
  description = "GitHub personal access token for private repos"
  sensitive   = true
}

variable "ami_name_prefix" {
  type    = string
  default = "groundhog-chuckd"
}

# Data source to get the latest Ubuntu 22.04 LTS AMI
data "amazon-ami" "ubuntu" {
  filters = {
    name                = "ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"
    root-device-type    = "ebs"
    virtualization-type = "hvm"
  }
  most_recent = true
  owners      = ["099720109477"] # Canonical
  region      = var.aws_build_region
}

# Build configuration
source "amazon-ebs" "ubuntu" {
  ami_name      = "${var.ami_name_prefix}-{{timestamp}}"
  instance_type = var.instance_type
  region        = var.aws_build_region
  source_ami    = data.amazon-ami.ubuntu.id
  ssh_username  = "ubuntu"

  # AMI configuration
  ami_description = "Groundhog AMI"

  # Add tags
  tags = {
    Name        = "${var.ami_name_prefix}-{{timestamp}}"
    Environment = "production"
    Application = "groundhog-chuckd"
    Built_With  = "Packer"
    Build_Date  = "{{timestamp}}"
  }

  # Snapshot tags
  snapshot_tags = {
    Name        = "${var.ami_name_prefix}-snapshot-{{timestamp}}"
    Application = "groundhog-chuckd"
  }

  # Instance configuration
  associate_public_ip_address = true
  ssh_timeout                 = "5m"

  # Use a larger root volume if needed
  launch_block_device_mappings {
    device_name           = "/dev/sda1"
    volume_size           = 20
    volume_type           = "gp3"
    delete_on_termination = true
  }

  # Configure AMI copying to additional regions
  ami_regions = var.ami_regions

  # Copy tags to additional regions
  copy_tags = true
}

# Build steps
build {
  name    = "groundhog-chuckd"
  sources = ["source.amazon-ebs.ubuntu"]

  # Wait for cloud-init to complete
  provisioner "shell" {
    inline = [
      "echo 'Waiting for cloud-init to complete...'",
      "cloud-init status --wait"
    ]
  }

  # Upload and execute your provisioning script
  provisioner "file" {
    source      = "./provision-script.sh"
    destination = "/tmp/provision-script.sh"
  }

  provisioner "shell" {
    environment_vars = [
      "GITHUB_TOKEN=${var.github_token}",
    ]
    inline = [
      "chmod +x /tmp/provision-script.sh",
      "sudo -E /tmp/provision-script.sh"
    ]
  }

  # Cleanup and prepare AMI
  provisioner "shell" {
    inline = [
      "echo 'Cleaning up...'",
      "sudo apt-get clean",
      "sudo apt-get autoremove -y",
      "sudo rm -rf /var/lib/apt/lists/*",
      "sudo rm -rf /tmp/*",
      "sudo rm -rf /var/tmp/*",
      "sudo rm -f /var/log/groundhog-build-script.log",
      "history -c && history -w",
      "sudo rm -f /root/.bash_history",
      "sudo rm -f /home/ubuntu/.bash_history"
    ]
  }

  # Verify installation
  provisioner "shell" {
    inline = [
      "echo 'Verifying chuckd installation...'",
      "which chuckd || (echo 'chuckd not found in PATH' && exit 1)",
      "chuckd --version || echo 'chuckd version check completed'"
    ]
  }
}
