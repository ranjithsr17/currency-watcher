variable "aws_region" {
  description = "AWS region for Currency Watcher"
  type        = string
  default     = "ap-south-1"
}

variable "project_name" {
  description = "Project name used for AWS resources"
  type        = string
  default     = "currency-watcher"
}

variable "environment" {
  description = "Deployment environment"
  type        = string
  default     = "production"
}

variable "container_port" {
  description = "Port exposed by the Go API"
  type        = number
  default     = 8080
}

variable "container_image" {
  description = "Docker image URI used by ECS"
  type        = string
}

variable "desired_count" {
  description = "Number of ECS tasks"
  type        = number
  default     = 0
}

variable "allowed_origins" {
  description = "Allowed frontend origins for CORS"
  type        = string
  default     = "http://localhost:5173"
}

