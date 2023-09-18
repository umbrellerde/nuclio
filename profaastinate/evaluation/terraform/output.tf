output "test_server_ip" {
  value = google_compute_instance.default.network_interface.0.access_config.0.nat_ip
}

output "test_server_ssh_login" {
  value = "ssh ubuntu@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}"
  description = "run this command to log into the server"
}

output "test_server_ssh_forward" {
  value = "ssh -N -L 9999:localhost:9090 -L 8070:localhost:8070 ubuntu@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}"
  description = "run this command to be able to access the gui of nuclio and minio from localhost (i.e., you can go to localhost:9000 to open the remotely running nuclio gui)"
}

output "test_server_sshfs" {
  value = "sshfs ubuntu@${google_compute_instance.default.network_interface.0.access_config.0.nat_ip}:/home/ ~/mount"
}