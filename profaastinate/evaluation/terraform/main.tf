terraform {
  required_providers {
    google = {
      source = "hashicorp/google"
      version = "4.51.0"
    }
  }
}

provider "google" {
  project = "schirmer-project"
  region = "europe-west3"
  zone = "europe-west3-a"

  credentials = file(var.key_file_location)
}

resource "google_compute_network" "vpc_network" {
  name = "profaastinate-network"
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "default" {
  ip_cidr_range = "10.0.1.0/24"
  name          = "profaastinate-subnetwork"
  network       = google_compute_network.vpc_network.id
}

resource "google_compute_instance" "default" {
  machine_type = "c3-standard-4" # "e2-standard-2"
  name         = "default-vm"
  zone = "europe-west3-a"
  tags = ["ssh"]

  boot_disk {
    initialize_params {
      image = "debian-cloud/debian-12"
    }
  }

  //metadata_startup_script = file("../deployment_scripts/installDeps.sh")

  metadata = {
    ssh-keys="debian:${file(var.gce_ssh_pub_key_file)}\ndebian:${file(var.gce_ssh_pub_raspi_key_file)}"
  }

  network_interface {
    subnetwork = google_compute_subnetwork.default.id
    access_config {
    }
  }
}

resource "google_compute_firewall" "ssh" {
  name = "allow-ssh"
  allow {
    protocol = "tcp"
    ports = ["22"]
  }
  direction = "INGRESS"
  network = google_compute_network.vpc_network.id
  source_ranges = ["0.0.0.0/0"]
  target_tags = ["ssh"]
}