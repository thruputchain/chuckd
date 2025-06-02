terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
  required_version = ">= 1.2.0"
}

# Variables for the deployment
variable "regions" {
  description = "EC2 regions"
  type        = list(string)
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.micro"
}

variable "config_files" {
  description = "Map of region to config file path"
  type        = map(string)
}

variable "ami_ids" {
  description = "Map of region to AMI ID"
  type        = map(string)
}

variable "metrics_port" {
  description = "Port number for metrics"
  type        = number
}

variable "app_port" {
  description = "Port number for the application"
  type        = number
}

variable "chuckd_args" {
  description = "Map of region to command line arguments for chuckd"
  type        = map(string)
  default     = {}
}

# Static provider configuration for each region
provider "aws" {
  alias  = "us_east_1"
  region = "us-east-1"
}

provider "aws" {
  alias  = "us_west_1"
  region = "us-west-1"
}

provider "aws" {
  alias  = "ap_northeast_1"
  region = "ap-northeast-1"
}

provider "aws" {
  alias  = "eu_central_1"
  region = "eu-central-1"
}