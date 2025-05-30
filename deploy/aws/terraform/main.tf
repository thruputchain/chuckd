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
  description = "List of AWS regions to deploy to"
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

# Create dynamic AWS providers for each region
locals {
  provider_aliases = {
    for region in var.regions :
    replace(region, "-", "_") => region
  }
}

# Dynamic provider configuration for all regions
provider "aws" {
  for_each = local.provider_aliases
  alias    = each.key
  region   = each.value
} 