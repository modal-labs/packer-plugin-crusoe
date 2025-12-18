# Packer Plugin for Crusoe - Examples

This directory contains example Packer templates for the Crusoe plugin.

## Prerequisites

1. Set up your Crusoe credentials:
   ```bash
   export CRUSOE_ACCESS_KEY_ID="your-access-key-id"
   export CRUSOE_SECRET_ACCESS_KEY="your-secret-access-key"
   ```

2. Install the Packer plugin (if not using `packer init`):
   ```bash
   cd ..
   make install
   ```

## Examples

### basic.pkr.hcl

A basic example that creates a custom Ubuntu image with nginx installed.

**Usage:**
```bash
cd examples
packer init basic.pkr.hcl
packer build basic.pkr.hcl
```

This example:
- Creates a temporary SSH key pair automatically
- Spins up an Ubuntu 22.04 instance
- Installs nginx and other packages
- Creates a custom image

### with-ssh-key.pkr.hcl

An example that uses an existing SSH key.

**Usage:**
1. Update the `ssh_key_ids` with your actual SSH key ID from Crusoe
2. Update the `ssh_private_key_file` path to your local SSH private key
3. Create a `setup-script.sh` file in this directory
4. Run:
   ```bash
   packer init with-ssh-key.pkr.hcl
   packer build with-ssh-key.pkr.hcl
   ```

This example:
- Uses an existing SSH key
- Uploads a custom script
- Executes the script on the instance

## Customization

You can customize these examples by:

1. **Changing the instance type**: Update the `instance_type` field (e.g., "h100-80gb.1x", "a40.2x")
2. **Changing the location**: Update the `location` field (e.g., "us-east1-a", "eu-west1-a")
3. **Using different base images**: Update the `image_id` field
4. **Adjusting disk size**: Modify the `disk_size_gib` field
5. **Adding more provisioners**: Add shell, file, or other provisioners to the build block

## Troubleshooting

### Authentication Errors
Make sure your credentials are correctly set in the environment variables or specified in the template.

### Timeout Errors
If your instance takes longer to boot, increase the `state_timeout` value:
```hcl
state_timeout = "20m"
```

### SSH Connection Issues
- Verify the SSH username is correct for your base image
- Check that the instance has a public IP address or you're using a private IP with network access
- Ensure the security group allows SSH access

## Additional Resources

- [Crusoe Cloud API Documentation](https://docs.crusoecloud.com/api/)
- [Packer Documentation](https://www.packer.io/docs)
- [Plugin Documentation](../docs/builders/crusoe.mdx)

