terraform {
  required_version = ">= 1.1.8, < 2.0.0"
  required_providers {
    ec = {
      source  = "elastic/ec"
      version = "0.5.1"
    }
  }
}

provider "ec" {}

module "tags" {
  source  = "../../infra/terraform/modules/tags"
  project = "apm-server"
}

locals {
  ci_tags = {
    environment  = coalesce(var.ENVIRONMENT, "dev")
    repo         = coalesce(var.REPO, "apm-server")
    branch       = var.BRANCH
    build        = var.BUILD_ID
    created_date = var.CREATED_DATE
    subproject   = "smoke-test"
  }
}

module "ec_deployment" {
  source = "../../infra/terraform/modules/ec_deployment"
  region = var.region

  deployment_template    = "gcp-cpu-optimized"
  deployment_name_prefix = "smoke-upgrade"

  apm_server_size = "1g"

  elasticsearch_size       = "4g"
  elasticsearch_zone_count = 1

  stack_version       = var.stack_version
  integrations_server = var.integrations_server
  tags                = merge(local.ci_tags, module.tags.tags)
}

variable "stack_version" {
  # By default match the latest available 7.17.x
  default     = "7.17.[0-9]?([0-9])$"
  description = "Optional stack version"
  type        = string
}

variable "integrations_server" {
  default     = true
  description = "Optionally disable the integrations server block and use the apm block (7.x only)"
  type        = bool
}

variable "region" {
  default     = "gcp-us-west2"
  description = "Optional ESS region where to run the smoke tests"
  type        = string
}

output "apm_secret_token" {
  value       = module.ec_deployment.apm_secret_token
  description = "The APM Server secret token"
  sensitive   = true
}

output "apm_server_url" {
  value       = module.ec_deployment.apm_url
  description = "The APM Server URL"
}

output "kibana_url" {
  value       = module.ec_deployment.kibana_url
  description = "The Kibana URL"
}

output "elasticsearch_url" {
  value       = module.ec_deployment.elasticsearch_url
  description = "The Elasticsearch URL"
}

output "elasticsearch_username" {
  value       = module.ec_deployment.elasticsearch_username
  sensitive   = true
  description = "The Elasticsearch username"
}

output "elasticsearch_password" {
  value       = module.ec_deployment.elasticsearch_password
  sensitive   = true
  description = "The Elasticsearch password"
}

output "stack_version" {
  value       = module.ec_deployment.stack_version
  description = "The matching stack pack version from the provided stack_version"
}

# CI variables
variable "BRANCH" {
  description = "Branch name or pull request for tagging purposes"
  default     = "unknown"
}

variable "BUILD_ID" {
  description = "Build ID in the CI for tagging purposes"
  default     = "unknown"
}

variable "CREATED_DATE" {
  description = "Creation date in epoch time for tagging purposes"
  default     = "unknown"
}

variable "ENVIRONMENT" {
  default = "unknown"
}

variable "REPO" {
  default = "unknown"
}