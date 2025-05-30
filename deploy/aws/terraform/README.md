# Chuckd Multi-Region AWS Deployment

This Terraform configuration automates the deployment of Chuckd servers across multiple AWS regions. It sets up the necessary infrastructure including VPCs, security groups, and Route 53 DNS records for service discovery.

## Prerequisites

1. [Terraform](https://www.terraform.io/downloads.html) installed (version >= 1.2.0)
2. AWS credentials configured (either via environment variables or AWS CLI configuration)
3. AMI images with Chuckd binary in each target region
4. Configuration files for each Chuckd instance

## Network Architecture

The deployment creates a sophisticated multi-region network setup:

1. **VPC Per Region**:
   - Each region gets its own VPC with a unique CIDR range (10.X.0.0/16)
   - Subnets are created with CIDR ranges 10.X.1.0/24

2. **VPC Peering**:
   - All regional VPCs are connected via VPC peering
   - Traffic between instances flows through AWS's internal network
   - Route tables are configured to route inter-VPC traffic through peering connections

3. **DNS Resolution**:
   - Private Route53 hosted zone for internal service discovery
   - DNS records use private IP addresses for optimal routing
   - Zone is associated with all regional VPCs

4. **Security Groups**:
   - Allow traffic on application and metrics ports from:
     - Public internet (for external access)
     - All VPC CIDR ranges (for internal communication)

## Configuration

1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Edit `terraform.tfvars` to specify:
   - AMI IDs for each region
   - Paths to configuration files for each instance
   - Port numbers for metrics and application
   - Instance type

## Directory Structure

```
terraform/
├── main.tf           # Provider configuration and variables
├── instances.tf      # Infrastructure resources
├── terraform.tfvars  # Your configuration values
└── configs/          # Directory for your Chuckd config files
    ├── us-east-1.json
    ├── us-west-1.json
    ├── ap-northeast-1.json
    └── eu-central-1.json
```

## Deployment Steps

1. Initialize Terraform:
   ```bash
   terraform init
   ```

2. Review the deployment plan:
   ```bash
   terraform plan
   ```

3. Apply the configuration:
   ```bash
   terraform apply
   ```

## DNS Names

Each instance will be accessible via a predictable DNS name in the format:
- `chuckd-{region}.chuckd.internal`

For example:
- `chuckd-us-east-1.chuckd.internal`
- `chuckd-us-west-1.chuckd.internal`
- `chuckd-ap-northeast-1.chuckd.internal`
- `chuckd-eu-central-1.chuckd.internal`

These DNS names resolve to private IP addresses within the VPC network, ensuring that inter-server communication uses AWS's internal network for optimal performance.

## Inter-Region Communication

When servers communicate with each other:
1. They use the internal DNS names (e.g., `chuckd-us-west-1.chuckd.internal`)
2. DNS resolves to the server's private IP address
3. Traffic is routed through VPC peering connections
4. Communication stays within AWS's network for better performance and lower latency

## Security Groups

The configuration creates security groups that:
- Allow inbound traffic on the specified metrics port from both internet and VPC networks
- Allow inbound traffic on the specified application port from both internet and VPC networks
- Allow all outbound traffic

## Cleanup

To destroy the infrastructure:
```bash
terraform destroy
```

Note: Destroying the infrastructure will remove all VPC peering connections and associated routes. Make sure all applications are properly shut down before destroying. 