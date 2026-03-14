# Packer Plugin for Crusoe Cloud

A [Packer](https://www.packer.io/) plugin that enables automated creation of custom VM images on [Crusoe Cloud](https://crusoecloud.com).

## Features

- **Automated Image Building**: Create custom VM images from base Crusoe Cloud images
- **Flexible SSH Authentication**: Auto-generates temporary SSH keys or uses your existing keys
- **Multi-Region Support**: Build images across different Crusoe Cloud locations
- **HMAC-Based Authentication**: Secure API communication using HMAC-SHA256 signatures
- **Automatic Cleanup**: Temporary resources are cleaned up automatically

## Quick Start

### Installation

```bash
# Install published releases with packer init / packer build
packer init image.pkr.hcl

# Build from source
make build

# Install to Packer plugins directory (typically '$HOME/.config/packer/plugins')
make install
```

Tagged `vX.Y.Z` releases are published automatically to GitHub Releases by
GitHub Actions so Packer can download this plugin directly from the repository.

Or use the install script:
```bash
./install.sh
```

### Basic Usage

Create a Packer template file (e.g., `image.pkr.hcl`):

```hcl
packer {
  required_plugins {
    crusoe = {
      version = ">= 1.0.0"
      source  = "github.com/modal-labs/crusoe"
    }
  }
}

source "crusoe" "ubuntu" {
  access_key_id     = "${env("CRUSOE_ACCESS_KEY_ID")}"
  secret_access_key = "${env("CRUSOE_SECRET_ACCESS_KEY")}"
  project_id        = "${env("CRUSOE_PROJECT_ID")}"
  
  location      = "us-northcentral1-a"
  instance_type = "a40.1x"
  image_id      = "ubuntu22.04:latest"
  
  image_name        = "my-custom-image"
  image_description = "Ubuntu with custom packages"
  
  ssh_username = "root"
}

build {
  sources = ["source.crusoe.ubuntu"]
  
  provisioner "shell" {
    inline = [
      "apt-get update",
      "apt-get install -y nginx vim curl"
    ]
  }
}
```

Build the image:

```bash
export CRUSOE_ACCESS_KEY_ID="your-access-key"
export CRUSOE_SECRET_ACCESS_KEY="your-secret-key"
export CRUSOE_PROJECT_ID="your-project-id"

packer build image.pkr.hcl
```

## Configuration

### Required Parameters

- `access_key_id` - Crusoe access key ID (env: `CRUSOE_ACCESS_KEY_ID`)
- `secret_access_key` - Crusoe secret access key (env: `CRUSOE_SECRET_ACCESS_KEY`)
- `project_id` - Crusoe project ID (env: `CRUSOE_PROJECT_ID`)
- `location` - Instance location (e.g., "us-northcentral1-a", "eu-iceland1-a")
- `instance_type` - VM type (e.g., "a40.1x", "c1a.32x")
- `image_id` - Base image ID (e.g., "ubuntu22.04:latest", "ubuntu24.04")

### Optional Parameters

- `api_endpoint` - API endpoint (default: "https://api.crusoecloud.com")
- `image_name` - Output image name (default: "packer-{timestamp}")
- `image_description` - Output image description
- `instance_name` - Temporary instance name (default: "packer-{timestamp}")
- `disk_size_gib` - Root disk size in GiB (default: 50)
- `instance_timeout` - Timeout for instance creation and operations (default: "20m")
- `image_timeout` - Timeout for image creation (default: "45m")
- `state_timeout` - **Deprecated**: Use `instance_timeout` instead
- `ssh_key_id` - Pre-existing SSH key ID to use
- `ssh_private_key_file` - Path to SSH private key file
- `ssh_username` - SSH username (default: "root")

For more information on how to configure the plugin, please read the
documentation located in the [`docs/`](docs) directory.

## Development

```bash
# Format code
make fmt

# Run tests
make test

# Generate code
make generate

# Clean build artifacts
make clean
```

## How It Works

1. **SSH Key Setup**: Creates temporary SSH key or uses provided key
2. **Instance Creation**: Launches VM instance with specified configuration
3. **Provisioning**: Runs your provisioner scripts (shell, file, etc.)
4. **Shutdown**: Gracefully stops the instance
5. **Image Creation**: Creates custom image from instance disk
6. **Cleanup**: Removes temporary resources (instance, SSH keys)

## Examples

See the `examples/` and `build/` directories for more examples:
- `examples/basic.pkr.hcl` - Simple single-region build
- `examples/with-ssh-key.pkr.hcl` - Multi-region build with custom SSH keys

## Requirements

- Go 1.24.0 or later
- Packer 1.7.0 or later
- Active Crusoe Cloud account with API credentials

## License

This project is maintained by Modal Labs.

## Links

- [Crusoe Cloud](https://crusoecloud.com)
- [Packer Documentation](https://www.packer.io/docs)
- [Crusoe API Documentation](https://docs.crusoecloud.com)
