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
  region = "europe-west4"
  zone = "europe-west4-a"

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
  region = "europe-west4"
}

resource "google_compute_instance" "default" {
  machine_type = "e2-highmem-8" # "e2-standard-2" # e2-highcpu-16 # c3-standard-8 # e2-stan-8 is 28ct/h
  name         = "default-vm"
  zone = "europe-west4-a"
  tags = ["ssh"]
  allow_stopping_for_update = true

  boot_disk {
    initialize_params {
      image = "ubuntu-os-cloud/ubuntu-2204-lts"
      size = 40
    }
  }

  // metadata_startup_script = "sudo apt-get update && sudo apt-get upgrade -y && sudo apt-get install -y sysstat htop git ca-certificates curl gnupg make golang-go && curl -fsSL https://get.docker.com -o get-docker.sh && sudo sh get-docker.sh && sudo usermod -aG docker $USER && sudo gpg -k && sudo gpg --no-default-keyring --keyring /usr/share/keyrings/k6-archive-keyring.gpg --keyserver hkp://keyserver.ubuntu.com:80 --recv-keys C5AD17C747E3415A3642D57D77C6C491D6AC1D69 && echo "deb [signed-by=/usr/share/keyrings/k6-archive-keyring.gpg] https://dl.k6.io/deb stable main" | sudo tee /etc/apt/sources.list.d/k6.list && sudo apt-get update && sudo apt-get install k6 && git clone https://github.com/umbrellerde/nuclio && cd nuclio && git switch 1.11.x && cd .. && touch done.txt"

  //metadata_startup_script = file("../deployment_scripts/installDeps.sh")

  metadata = {
    ssh-keys="ubuntu:${file(var.gce_ssh_pub_key_file)}\nubuntu:${file(var.gce_ssh_pub_raspi_key_file)}"
  }

  network_interface {
    subnetwork = google_compute_subnetwork.default.id
    access_config {
    }
  }

}

resource "google_compute_firewall" "ssh" {
  name = "allow-ssh-profaastinate"
  allow {
    protocol = "tcp"
    ports = ["22"]
  }
  direction = "INGRESS"
  network = google_compute_network.vpc_network.id
  source_ranges = ["0.0.0.0/0"]
  target_tags = ["ssh"]
}