packer {
  required_plugins {
    crusoe = {
      version = ">= 0.0.1"
      source  = "github.com/modal-labs/crusoe"
    }
  }
}

variable "crusoe_access_key_id" {
  type    = string
  default = "${env("CRUSOE_ACCESS_KEY_ID")}"
}

variable "crusoe_secret_access_key" {
  type      = string
  default   = "${env("CRUSOE_SECRET_ACCESS_KEY")}"
  sensitive = true
}

source "crusoe" "ubuntu" {
  access_key_id     = "${var.crusoe_access_key_id}"
  secret_access_key = "${var.crusoe_secret_access_key}"

  location      = "us-northcentral1-a"
  instance_type = "a40.1x"
  image_id      = "ubuntu22.04:latest"

  image_name        = "my-custom-ubuntu-image"
  image_description = "Ubuntu 22.04 with custom configuration"

  disk_size_gib = 100
  state_timeout = "15m"

  ssh_username = "root"
  skip_publish = true
}

build {
  sources = ["source.crusoe.ubuntu"]

  provisioner "shell" {
    inline = [
      "apt-get update",
      "apt-get install -y nginx curl vim",
      "systemctl enable nginx"
    ]
  }

  provisioner "shell" {
    inline = [
      "echo 'Custom image created at' $(date) > /etc/image-build-info"
    ]
  }
}
