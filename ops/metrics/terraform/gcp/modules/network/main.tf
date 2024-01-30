resource "google_compute_network" "vpc_network" {
  name                    = "${var.region}-bacalhau-otel-vpc-network"
  auto_create_subnetworks = var.auto_subnets
}

resource "google_compute_subnetwork" "subnetwork" {
  name          = "${var.region}-bacalhau-otel-subnetwork"
  ip_cidr_range = var.subnet_cidr
  region        = var.region
  network       = google_compute_network.vpc_network.name
}

resource "google_compute_address" "otel_collector_ip" {
  name   = "otel-collector-ip"
  region = var.region
}

resource "google_compute_firewall" "google_firewall_egress" {
  name    = "bacalhau-otel-firewall-egress"
  network = google_compute_network.vpc_network.name

  direction = "EGRESS"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = var.egress_tcp_ports
  }

  source_ranges = var.egress_source_ranges
}

resource "google_compute_firewall" "bacalhau_protocol_firewall_ingress" {
  name    = "bacalhau-otel-firewall-ingress"
  network = google_compute_network.vpc_network.name

  direction = "INGRESS"

  allow {
    protocol = "icmp"
  }

  allow {
    protocol = "tcp"
    ports    = var.ingress_tcp_ports
  }

  source_ranges = var.ingress_source_ranges
}
