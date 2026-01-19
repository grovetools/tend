package demo

// This file contains template content constants used by demo specifications.
// These constants provide realistic file contents for various languages and tools.

const tsConfigContent = `{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "jsx": "react-jsx",
    "strict": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "resolveJsonModule": true,
    "isolatedModules": true,
    "noEmit": true,
    "skipLibCheck": true,
    "esModuleInterop": true,
    "allowSyntheticDefaultImports": true,
    "forceConsistentCasingInFileNames": true
  },
  "include": ["src"]
}
`

const catppuccinTheme = `/* Catppuccin Mocha Theme for Dashboard */
:root {
  --ctp-rosewater: #f5e0dc;
  --ctp-flamingo: #f2cdcd;
  --ctp-pink: #f5c2e7;
  --ctp-mauve: #cba6f7;
  --ctp-red: #f38ba8;
  --ctp-maroon: #eba0ac;
  --ctp-peach: #fab387;
  --ctp-yellow: #f9e2af;
  --ctp-green: #a6e3a1;
  --ctp-teal: #94e2d5;
  --ctp-sky: #89dceb;
  --ctp-sapphire: #74c7ec;
  --ctp-blue: #89b4fa;
  --ctp-lavender: #b4befe;
  --ctp-text: #cdd6f4;
  --ctp-subtext1: #bac2de;
  --ctp-subtext0: #a6adc8;
  --ctp-overlay2: #9399b2;
  --ctp-overlay1: #7f849c;
  --ctp-overlay0: #6c7086;
  --ctp-surface2: #585b70;
  --ctp-surface1: #45475a;
  --ctp-surface0: #313244;
  --ctp-base: #1e1e2e;
  --ctp-mantle: #181825;
  --ctp-crust: #11111b;
}

body {
  background-color: var(--ctp-base);
  color: var(--ctp-text);
  font-family: system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
}

.widget {
  background-color: var(--ctp-surface0);
  border: 1px solid var(--ctp-surface1);
  border-radius: 8px;
  padding: 1rem;
}

.widget-title {
  color: var(--ctp-mauve);
  font-weight: 600;
}
`

const terraformMain = `# Homelab Infrastructure

terraform {
  required_version = ">= 1.5.0"

  required_providers {
    proxmox = {
      source  = "telmate/proxmox"
      version = "~> 2.9"
    }
    cloudflare = {
      source  = "cloudflare/cloudflare"
      version = "~> 4.0"
    }
  }
}

provider "proxmox" {
  pm_api_url      = var.proxmox_api_url
  pm_user         = var.proxmox_user
  pm_password     = var.proxmox_password
  pm_tls_insecure = true
}

module "kubernetes" {
  source = "./modules/kubernetes"

  node_count    = var.k8s_node_count
  node_memory   = var.k8s_node_memory
  node_cores    = var.k8s_node_cores
  storage_pool  = var.storage_pool
}

module "storage" {
  source = "./modules/storage"

  nfs_enabled = var.nfs_enabled
  ceph_nodes  = var.ceph_nodes
}
`

const terraformVars = `variable "proxmox_api_url" {
  description = "Proxmox API endpoint URL"
  type        = string
}

variable "proxmox_user" {
  description = "Proxmox API user"
  type        = string
  default     = "root@pam"
}

variable "proxmox_password" {
  description = "Proxmox API password"
  type        = string
  sensitive   = true
}

variable "k8s_node_count" {
  description = "Number of Kubernetes nodes"
  type        = number
  default     = 3
}

variable "k8s_node_memory" {
  description = "Memory per node in MB"
  type        = number
  default     = 4096
}

variable "k8s_node_cores" {
  description = "CPU cores per node"
  type        = number
  default     = 2
}

variable "storage_pool" {
  description = "Proxmox storage pool"
  type        = string
  default     = "local-lvm"
}

variable "nfs_enabled" {
  description = "Enable NFS storage"
  type        = bool
  default     = true
}

variable "ceph_nodes" {
  description = "Ceph storage nodes"
  type        = list(string)
  default     = []
}
`

const ansiblePlaybook = `---
- name: Configure homelab infrastructure
  hosts: all
  become: true

  vars:
    docker_version: "24.0"
    k3s_version: "v1.28.5+k3s1"

  roles:
    - role: common
      tags: [base]
    - role: docker
      tags: [docker]
    - role: k3s
      tags: [kubernetes]
      when: inventory_hostname in groups['k3s_nodes']
    - role: monitoring
      tags: [monitoring]
      when: inventory_hostname in groups['monitoring']
`

const helmChart = `apiVersion: v2
name: homelab
description: Helm charts for homelab services
type: application
version: 0.1.0
appVersion: "1.0.0"

dependencies:
  - name: prometheus
    version: "25.x.x"
    repository: https://prometheus-community.github.io/helm-charts
    condition: prometheus.enabled
  - name: grafana
    version: "7.x.x"
    repository: https://grafana.github.io/helm-charts
    condition: grafana.enabled
`

const helmValues = `# Default values for homelab chart

replicaCount: 1

image:
  repository: ghcr.io/homelab
  pullPolicy: IfNotPresent
  tag: ""

service:
  type: ClusterIP
  port: 80

ingress:
  enabled: true
  className: traefik
  annotations:
    cert-manager.io/cluster-issuer: letsencrypt-prod
  hosts:
    - host: homelab.local
      paths:
        - path: /
          pathType: Prefix

prometheus:
  enabled: true

grafana:
  enabled: true
  adminPassword: changeme
`
