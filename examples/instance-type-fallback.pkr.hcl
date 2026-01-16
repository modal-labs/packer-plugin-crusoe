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

  location       = "us-east1-a"
  instance_types = ["a40.1x", "a100-80gb-sxm-ib.8x", "c1a.32x"]
  image_id       = "ubuntu22.04:latest"

  api_call_retries = 3

  ssh_key_id = "global-worker"

  image_name        = "custom-test-image"
  image_description = "Custom test image"
  ssh_username      = "root"
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

