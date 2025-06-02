# Example configuration file - copy to terraform.tfvars and fill in your values

# List of AWS regions to deploy to
regions = [
  "us-east-1",
  "us-west-1",
  "ap-northeast-1",
  "eu-central-1"
]

# AMI IDs for each region (replace with your actual AMI IDs)
ami_ids = {
  "us-east-1"      = "ami-070dbd17925473767"
  "us-west-1"      = "ami-0270d7b37f2ed8ca7"
  "ap-northeast-1" = "ami-0a9c3e5d2c1ddd3cf"
  "eu-central-1"   = "ami-0bbbbe27e5904cfcf"
}

# Paths to config files for each region
config_files = {
  "us-east-1"      = "configs/us-east-1.json"
  "us-west-1"      = "configs/us-west-1.json"
  "ap-northeast-1" = "configs/ap-northeast-1.json"
  "eu-central-1"   = "configs/eu-central-1.json"
}

# Command line arguments for each region
chuckd_args = {
  "us-east-1"      = "-v --config /etc/chuckd/config.json --instance leader --seed 8502489525653699386 --metrics :2222 solo leader --addr :4000 --delay 100"
  "us-west-1"      = "-v --config /etc/chuckd/config.json --instance quorum-0 --seed 6311280341148717924 --metrics :2222 bench testcluster --tps 1000 --out_addr :4000 --w_addr :8383"
  "ap-northeast-1" = "-v --config /etc/chuckd/config.json --instance quorum-1 --seed 2622264106489803354 --metrics :2222 bench testcluster --tps 1000 --out_addr :4000 --w_addr :8383"
  "eu-central-1"   = "-v --config /etc/chuckd/config.json --instance quorum-2 --seed 7964169681022589069 --metrics :2222 bench testcluster --tps 1000 --out_addr :4000 --w_addr :8383"
}

# Port configurations
metrics_port = 2222
app_port     = 4000

# Instance type
instance_type = "t3.large"