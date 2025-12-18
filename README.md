# Packer Plugin for Crusoe Cloud

The Crusoe Packer plugin provides a builder for creating custom images on [Crusoe Cloud](https://crusoecloud.com).

## Installation

### Using Pre-built Releases

#### Using the `packer init` command

Starting from version 1.7, Packer supports a new `packer init` command allowing
automatic installation of Packer plugins. Read the
[Packer documentation](https://www.packer.io/docs/commands/init) for more information.

To install this plugin, copy and paste this code into your Packer configuration.
Then, run [`packer init`](https://www.packer.io/docs/commands/init).

```hcl
packer {
  required_plugins {
    crusoe = {
      version = ">= 1.0.0"
      source  = "github.com/modal-labs/crusoe"
    }
  }
}
```

#### Manual Installation

You can find pre-built binary releases of the plugin [here](https://github.com/crusoecloud/packer-plugin-crusoe/releases).
Once you have downloaded the latest archive corresponding to your target OS,
uncompress it to retrieve the plugin binary file corresponding to your platform.

To install the plugin, please follow the Packer documentation on
[installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).

### From Sources

If you prefer to build the plugin from sources, clone the GitHub repository
locally and run the command `go build` from the root
directory. Upon successful compilation, a `packer-plugin-crusoe` plugin
binary file can be found in the root directory.

To install the compiled plugin, please follow the official Packer documentation
on [installing a plugin](https://www.packer.io/docs/extending/plugins/#installing-plugins).

### Configuration

For more information on how to configure the plugin, please refer to the
[builder documentation](docs/builders/crusoe.mdx).

## Usage

Here's a basic example of how to use the Crusoe builder:

```hcl
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

source "crusoe" "example" {
  access_key_id     = "${var.crusoe_access_key_id}"
  secret_access_key = "${var.crusoe_secret_access_key}"
  
  location      = "us-northcentral1-a"
  instance_type = "a40.1x"
  image_id      = "ubuntu22.04:latest"
  
  image_name        = "my-custom-image"
  image_description = "My custom image built with Packer"
  
  ssh_username = "root"
}

build {
  sources = ["source.crusoe.example"]
  
  provisioner "shell" {
    inline = [
      "apt-get update",
      "apt-get install -y nginx",
    ]
  }
}
```

## Authentication

The plugin supports the following methods for authentication:

1. **Environment Variables** (recommended):
   ```bash
   export CRUSOE_ACCESS_KEY_ID="your-access-key-id"
   export CRUSOE_SECRET_ACCESS_KEY="your-secret-access-key"
   ```

2. **Configuration File**: Specify the credentials directly in your Packer template.

## How It Works

The Crusoe builder performs the following steps:

1. **Create SSH Key** (if needed): If no SSH authentication is configured, a temporary SSH key pair is created and uploaded to Crusoe.

2. **Create Instance**: A new VM instance is created using the specified base image, instance type, and location.

3. **Wait for Instance**: The builder waits for the instance to become active and accessible via SSH.

4. **Provision**: Packer runs all configured provisioners (shell scripts, file uploads, etc.) on the instance.

5. **Shutdown**: The instance is gracefully shut down using SSH, then the builder ensures it's stopped via the API.

6. **Create Image**: A custom image is created from the instance's disk.

7. **Cleanup**: The temporary instance and SSH key (if created) are deleted.

The resulting custom image can be used to launch new instances on Crusoe Cloud.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

This plugin is released under the Mozilla Public License 2.0. See the [LICENSE](LICENSE) file for details.

