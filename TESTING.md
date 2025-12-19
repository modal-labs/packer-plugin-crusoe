# Testing Guide for Packer Plugin Crusoe

This document provides guidelines for testing the Packer Plugin for Crusoe Cloud.

## Unit Tests

The plugin includes unit tests for configuration validation and other core functionality.

### Running Unit Tests

```bash
# Run all tests
make test

# Run tests with verbose output
go test ./... -v

# Run tests for a specific package
go test ./builder/crusoe -v

# Run a specific test
go test ./builder/crusoe -v -run TestConfigPrepare_Required

# Run tests with coverage
go test ./... -cover

# Generate coverage report
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Integration Testing

To test the plugin against the actual Crusoe Cloud API, you'll need:

1. Valid Crusoe Cloud credentials
2. A valid location where you can create instances
3. A valid instance type available in that location
4. A valid base image ID

### Prerequisites

```bash
# Set up your credentials
export CRUSOE_ACCESS_KEY_ID="your-access-key-id"
export CRUSOE_SECRET_ACCESS_KEY="your-secret-access-key"

# Optional: Set a custom API endpoint
export CRUSOE_API_ENDPOINT="https://api.crusoecloud.com"
```

### Install the Plugin Locally

```bash
# Build and install the plugin
make install

# Or for development (places in ~/.packer.d/plugins/)
make dev
```

### Test with a Basic Build

1. Create a test HCL file:

```hcl
packer {
  required_plugins {
    crusoe = {
      version = ">= 1.0.0"
      source  = "github.com/modal-labs/crusoe"
    }
  }
}

source "crusoe" "test" {
  access_key_id     = "${env("CRUSOE_ACCESS_KEY_ID")}"
  secret_access_key = "${env("CRUSOE_SECRET_ACCESS_KEY")}"
  
  location      = "us-northcentral1-a"
  instance_type = "a40.1x"
  image_id      = "ubuntu22.04:latest"
  
  image_name = "test-image-${formatdate("YYYY-MM-DD-hhmmss", timestamp())}"
  
  ssh_username = "root"
}

build {
  sources = ["source.crusoe.test"]
  
  provisioner "shell" {
    inline = [
      "echo 'Testing provisioning'",
      "uname -a",
    ]
  }
}
```

2. Run the build:

```bash
# Initialize (if using packer init)
packer init test.pkr.hcl

# Validate the template
packer validate test.pkr.hcl

# Build
packer build test.pkr.hcl
```

### Debug Mode

For detailed debugging output:

```bash
# Enable debug mode with verbose logging
PACKER_LOG=1 packer build -debug test.pkr.hcl

# This will:
# - Show detailed API requests and responses
# - Pause between steps
# - Save the SSH key to disk for inspection
```

## Testing Checklist

When testing the plugin, verify the following functionality:

### Authentication
- [ ] Plugin accepts credentials from HCL config
- [ ] Plugin reads credentials from environment variables
- [ ] Plugin fails gracefully with invalid credentials

### Instance Creation
- [ ] Instance is created with correct specifications
- [ ] Instance reaches "running" state
- [ ] Instance has a public IP address (or private IP if no public IP)
- [ ] Temporary SSH key is created when no SSH credentials provided
- [ ] Existing SSH key works when provided

### SSH Connectivity
- [ ] SSH connection is established successfully
- [ ] SSH username is correctly configured
- [ ] SSH private key authentication works
- [ ] SSH connection timeout is respected

### Provisioning
- [ ] Shell provisioners execute successfully
- [ ] File provisioners upload files correctly
- [ ] Multiple provisioners can be chained
- [ ] Provisioner failures halt the build

### Shutdown
- [ ] Instance is shut down gracefully via SSH
- [ ] Instance state is updated to "stopped" via API
- [ ] Shutdown timeout is respected

### Image Creation
- [ ] Custom image is created from instance disk
- [ ] Image creation completes successfully
- [ ] Image has correct name and description
- [ ] Image is available in the specified location

### Cleanup
- [ ] Temporary instance is deleted after build
- [ ] Temporary SSH key is deleted after build
- [ ] Cleanup happens even if build fails
- [ ] No resources are leaked

### Error Handling
- [ ] Clear error messages for API failures
- [ ] Graceful handling of timeouts
- [ ] Proper cleanup on cancellation (Ctrl+C)
- [ ] Validation errors are caught early

## Mock Testing (Future Enhancement)

For testing without hitting the real API, consider implementing mock tests:

```go
// Example mock test structure
type mockClient struct {
    createInstanceFunc func(*CreateInstanceRequest) (*Instance, error)
    // ... other methods
}

func (m *mockClient) CreateInstance(req *CreateInstanceRequest) (*Instance, error) {
    return m.createInstanceFunc(req)
}

// Then in tests:
func TestStepCreateInstance(t *testing.T) {
    mockClient := &mockClient{
        createInstanceFunc: func(req *CreateInstanceRequest) (*Instance, error) {
            return &Instance{
                ID: "test-instance-id",
                // ... other fields
            }, nil
        },
    }
    // ... rest of test
}
```

## Troubleshooting Tests

### Tests Fail with "no such file or directory"
Make sure you're running tests from the project root:
```bash
cd /home/ec2-user/packer-plugin-crusoe
go test ./...
```

### Integration Tests Fail with Authentication Error
1. Check that environment variables are set correctly
2. Verify credentials are valid
3. Check API endpoint is correct

### Build Hangs During SSH Connection
1. Verify the instance has a public IP or you have network access to private IP
2. Check security groups allow SSH access
3. Verify SSH username is correct for the base image
4. Increase SSH timeout in configuration

### Instance Not Cleaning Up
If instances are left running after a failed build:
```bash
# Use Crusoe CLI or API to list and delete instances
crusoe compute instances list
crusoe compute instances delete <instance-id>
```

## Continuous Integration

For CI/CD pipelines, consider:

1. **Unit Tests Only**: Run unit tests in CI without credentials
   ```bash
   go test ./builder/crusoe -v
   ```

2. **Integration Tests**: Use a separate job with credentials stored securely
   ```yaml
   # Example for GitHub Actions
   - name: Run Integration Tests
     env:
       CRUSOE_ACCESS_KEY_ID: ${{ secrets.CRUSOE_ACCESS_KEY_ID }}
       CRUSOE_SECRET_ACCESS_KEY: ${{ secrets.CRUSOE_SECRET_ACCESS_KEY }}
     run: packer build examples/basic.pkr.hcl
   ```

3. **Test Image Cleanup**: Always clean up test images after integration tests
   ```bash
   # Script to delete test images older than 1 day
   crusoe compute custom-images list --filter "name:test-*" | \
     jq -r '.[] | select(.created < (now - 86400)) | .id' | \
     xargs -I {} crusoe compute custom-images delete {}
   ```

## Performance Testing

To test performance:

1. **Build Time**: Measure total time from start to image creation
2. **API Response Times**: Log time for each API call
3. **SSH Connection Time**: Measure time to establish SSH connection
4. **Image Creation Time**: Track how long snapshot takes

```bash
# Time a build
time packer build test.pkr.hcl
```

## Security Testing

1. **Credential Handling**: Verify credentials are not logged
2. **SSH Key Management**: Ensure temporary keys are deleted
3. **API Communication**: Verify HTTPS is used for all API calls
4. **Error Messages**: Check that sensitive data is not exposed in errors

## Additional Resources

- [Packer Testing Documentation](https://www.packer.io/docs/debugging)
- [Go Testing Documentation](https://golang.org/pkg/testing/)
- [Crusoe Cloud API Documentation](https://docs.crusoecloud.com/api/)




