variable "key_file_location" {
  description = "Path of the .json file containing the google credentials"
}

variable "gce_ssh_pub_key_file" {
  description = "Path of the public key file you will use to log in"
}

variable "gce_ssh_pub_raspi_key_file" {
  description = "well known semi-public ssh key so that other people can log in as well"
}