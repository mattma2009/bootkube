variable "access_key_id" {
  type = "string"
}

variable "access_key" {
  type = "string"
}

variable "ssh_key" {
  description = "aws ssh key"
  type        = "string"
}

variable "instance_tags" {
  description = "Name all instances behind a single tag based on who/what is running terraform"
  type        = "string"
}

variable "instance_type" {
  description = "Name all instances behind a single tag based on who/what is running terraform"
  type        = "string"
  default     = "m3.medium"
}

variable "self_host_etcd" {
  type    = "string"
  default = "true"
}

variable "calico_network_policy" {
  type    = "string"
  default = "true"
}

variable "num_workers" {
  description = "number of worker nodes"
  type        = "string"
  default     = "1"
}

variable "additional_masters" {
  description = "number of additional master nodes not including bootstrap node"
  type        = "string"
  default     = "0"
}

variable "region" {
  description = "aws region"
  type        = "string"
  default     = "us-east-1"
}
