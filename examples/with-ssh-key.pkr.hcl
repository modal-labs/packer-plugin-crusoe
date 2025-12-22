packer {
  required_plugins {
    crusoe = {
      version = ">= 1.0.0"
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

# Example with an existing SSH key
source "crusoe" "ubuntu-with-key" {
  access_key_id     = "${var.crusoe_access_key_id}"
  secret_access_key = "${var.crusoe_secret_access_key}"
  
  location      = "us-northcentral1-a"
  instance_type = "a40.1x"
  image_id      = "ubuntu22.04:latest"
  
  # Use an existing SSH key
  ssh_key_ids = "your-ssh-key-id-here"
  
  image_name        = "my-custom-image"
  image_description = "Custom image with SSH key"
  
  ssh_username         = "root"
  ssh_private_key_file = "~/.ssh/id_rsa"
}

build {
  sources = ["source.crusoe.ubuntu-with-key"]
  
  provisioner "file" {
    source      = "setup-script.sh"
    destination = "/tmp/setup-script.sh"
  }
  
  provisioner "shell" {
    inline = [
      "chmod +x /tmp/setup-script.sh",
      "/tmp/setup-script.sh"
    ]
  }
}

