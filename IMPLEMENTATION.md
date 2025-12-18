# Packer Plugin for Crusoe Cloud - Implementation Details

This document provides a comprehensive overview of the implementation of the Packer Plugin for Crusoe Cloud.

## Overview

The Packer Plugin for Crusoe Cloud enables automated creation of custom VM images on Crusoe Cloud infrastructure. The plugin follows Packer's plugin architecture and implements a custom builder that:

1. Creates a temporary VM instance
2. Provisions it using SSH-based provisioners
3. Shuts down the instance
4. Creates a custom image from the instance's disk
5. Cleans up temporary resources

## Architecture

### Plugin Structure

```
packer-plugin-crusoe/
├── main.go                          # Plugin entry point
├── builder/
│   └── crusoe/
│       ├── builder.go               # Builder implementation
│       ├── config.go                # Configuration and validation
│       ├── config.hcl2spec.go       # HCL2 spec (auto-generated)
│       ├── artifact.go              # Build artifact definition
│       ├── crusoe.go                # API client implementation
│       ├── step_create_ssh_key.go   # Step: Create temporary SSH key
│       ├── step_create_instance.go  # Step: Create VM instance
│       ├── step_shutdown.go         # Step: Shutdown instance
│       ├── step_create_image.go     # Step: Create custom image
│       ├── wait.go                  # State waiting utilities
│       └── config_test.go           # Unit tests
├── docs/
│   └── builders/
│       └── crusoe.mdx               # User documentation
├── examples/
│   ├── basic.pkr.hcl                # Basic example
│   ├── with-ssh-key.pkr.hcl         # Example with existing SSH key
│   └── README.md                    # Examples documentation
├── go.mod                           # Go module definition
├── go.sum                           # Go module checksums
├── Makefile                         # Build automation
├── README.md                        # Main documentation
├── TESTING.md                       # Testing guide
└── IMPLEMENTATION.md                # This file

```

## Component Details

### 1. main.go - Plugin Entry Point

The entry point registers the Crusoe builder with Packer's plugin system:

```go
pps := plugin.NewSet()
pps.RegisterBuilder(plugin.DEFAULT_NAME, new(crusoe.Builder))
pps.SetVersion(PluginVersion)
err := pps.Run()
```

### 2. Builder (builder.go)

Implements the `packer.Builder` interface with three main methods:

- **ConfigSpec()**: Returns the HCL2 specification for parsing configuration
- **Prepare()**: Validates and prepares the configuration
- **Run()**: Executes the build process using a multi-step runner

#### Build Steps

The builder uses Packer's multi-step pattern:

1. `stepCreateSSHKey`: Creates a temporary SSH key pair if needed
2. `StepConnect`: Establishes SSH connection to the instance
3. `StepProvision`: Runs all configured provisioners
4. `StepCleanupTempKeys`: Removes temporary SSH keys from instance
5. `stepShutdown`: Gracefully shuts down the instance
6. `stepCreateImage`: Creates a custom image from the disk

Each step implements:
- **Run()**: Execute the step
- **Cleanup()**: Clean up resources if the build fails or is cancelled

### 3. Configuration (config.go)

Defines all configuration options with validation:

#### Required Fields
- `access_key_id`: Crusoe API access key
- `secret_access_key`: Crusoe API secret key
- `location`: Location for the instance (e.g., "us-northcentral1-a")
- `instance_type`: Instance type (e.g., "a40.1x")
- `image_id`: Base image ID (e.g., "ubuntu22.04:latest")

#### Optional Fields
- `api_endpoint`: Custom API endpoint (default: https://api.crusoecloud.com)
- `network_id`: Network to attach to
- `subnet_id`: Subnet to attach to
- `ssh_key_ids`: List of existing SSH keys to use
- `instance_name`: Name for the temporary instance
- `image_name`: Name for the resulting image
- `image_description`: Description for the resulting image
- `disk_size_gib`: Root disk size (default: 50 GB)
- `state_timeout`: Timeout for waiting on state changes (default: 10m)
- `userdata`: User data to pass to the instance
- `tags`: Tags to apply to resources

#### Configuration Validation

The `Prepare()` method:
1. Decodes HCL configuration into the Config struct
2. Reads credentials from environment variables if not provided
3. Validates required fields
4. Sets default values for optional fields
5. Parses and validates durations
6. Validates SSH communicator configuration

### 4. API Client (crusoe.go)

Implements a REST API client for Crusoe Cloud:

#### Client Structure
```go
type Client struct {
    accessKeyID     string
    secretAccessKey string
    apiEndpoint     string
    httpClient      *http.Client
}
```

#### Authentication
Uses Bearer token authentication:
```
Authorization: Bearer {secretAccessKey}
```

#### API Methods

**Instance Management:**
- `CreateInstance()`: Creates a new VM instance
- `GetInstance()`: Retrieves instance details
- `UpdateInstance()`: Updates instance state (e.g., stop)
- `DeleteInstance()`: Deletes an instance

**Image Management:**
- `CreateCustomImage()`: Creates a custom image from a disk
- `GetCustomImage()`: Retrieves image details
- `DeleteCustomImage()`: Deletes a custom image

**SSH Key Management:**
- `CreateSSHKey()`: Uploads an SSH public key
- `DeleteSSHKey()`: Deletes an SSH key

#### API Endpoints

All endpoints use the base URL `https://api.crusoecloud.com/v1alpha5`:

- `POST /compute/instances` - Create instance
- `GET /compute/instances/{id}` - Get instance
- `PATCH /compute/instances/{id}` - Update instance
- `DELETE /compute/instances/{id}` - Delete instance
- `POST /compute/custom-images` - Create custom image
- `GET /compute/custom-images/{id}` - Get custom image
- `DELETE /compute/custom-images/{id}` - Delete custom image
- `POST /ssh-keys` - Create SSH key
- `DELETE /ssh-keys/{id}` - Delete SSH key

### 5. Build Steps Implementation

#### Step 1: Create SSH Key (step_create_ssh_key.go)

**Purpose:** Create a temporary SSH key pair if the user hasn't provided one.

**Logic:**
1. Check if SSH password or private key file is configured
2. If not, generate a 2048-bit RSA key pair
3. Upload the public key to Crusoe Cloud
4. Store the private key in the communicator config
5. If debug mode is enabled, save the private key to disk

**Cleanup:** Deletes the SSH key from Crusoe Cloud using the API.

#### Step 2: Create Instance (step_create_instance.go)

**Purpose:** Create a VM instance on Crusoe Cloud.

**Logic:**
1. Build the instance creation request with:
   - Instance name, type, location
   - Base image ID
   - SSH keys (including temporary key if created)
   - Network configuration
   - Disk configuration
   - User data and tags
2. Call the CreateInstance API
3. Wait for the instance to reach "running" state
4. Retrieve the instance's IP address (public or private)
5. Store instance info in the state bag for later steps

**Cleanup:** Deletes the instance if the build fails.

#### Step 3: Shutdown (step_shutdown.go)

**Purpose:** Gracefully shut down the instance before creating the image.

**Logic:**
1. Wait a short delay for any pending operations to complete
2. Attempt graceful shutdown via SSH: `sudo shutdown -h now`
3. Check if SSH command executed successfully
4. Call the UpdateInstance API to ensure instance is stopped
5. Wait for instance to reach "stopped" state

**Cleanup:** None (shutdown is idempotent).

#### Step 4: Create Image (step_create_image.go)

**Purpose:** Create a custom image from the instance's disk.

**Logic:**
1. Get the disk ID from the instance's disk attachments
2. Call the CreateCustomImage API with:
   - Image name and description
   - Location
   - Source disk ID
3. Wait for the image to reach "available" state
4. Store the image info in the state bag

**Cleanup:** None (image creation is the desired outcome).

### 6. State Waiting (wait.go)

Implements polling functions to wait for resources to reach desired states:

#### waitForInstanceState()
- Polls the instance status every 3 seconds
- Returns when instance reaches the target state
- Returns error if timeout is exceeded
- Respects cancellation via context

#### waitForImageState()
- Similar to waitForInstanceState but for images
- Waits for image to be created and available

### 7. Artifact (artifact.go)

Represents the final build output:

**Fields:**
- `ImageID`: The ID of the created image
- `ImageName`: The name of the created image
- `Description`: Image description
- `Location`: Location where image is stored

**Methods:**
- `BuilderId()`: Returns the builder ID for Packer
- `Id()`: Returns the image ID
- `String()`: Returns a human-readable description
- `Destroy()`: Deletes the custom image
- `State()`: Returns artifact state for HCP integration

## API Request/Response Examples

### Create Instance Request
```json
{
  "name": "packer-1234567890",
  "type": "a40.1x",
  "location": "us-northcentral1-a",
  "image": "ubuntu22.04:latest",
  "ssh_keys": ["ssh-key-id-123"],
  "disks": [{
    "size_gib": 50,
    "type": "persistent-ssd",
    "mode": "read-write"
  }]
}
```

### Create Instance Response
```json
{
  "instance": {
    "id": "instance-abc123",
    "name": "packer-1234567890",
    "type": "a40.1x",
    "location": "us-northcentral1-a",
    "state": "starting",
    "public_ipv4": {
      "address": "203.0.113.10"
    },
    "disk_attachments": [{
      "id": "disk-xyz789",
      "mode": "read-write"
    }]
  }
}
```

### Update Instance Request (Shutdown)
```json
{
  "state": "stopped"
}
```

### Create Custom Image Request
```json
{
  "name": "my-custom-ubuntu-image",
  "description": "Ubuntu 22.04 with custom configuration",
  "location": "us-northcentral1-a",
  "source_disk": "disk-xyz789"
}
```

### Create Custom Image Response
```json
{
  "image": {
    "id": "image-def456",
    "name": "my-custom-ubuntu-image",
    "description": "Ubuntu 22.04 with custom configuration",
    "state": "creating",
    "location": "us-northcentral1-a"
  }
}
```

## Error Handling

The plugin implements comprehensive error handling:

1. **Configuration Errors:** Validated early in the Prepare() phase
2. **API Errors:** Wrapped with context using fmt.Errorf with %w verb
3. **Timeout Errors:** Detected and reported with clear messages
4. **Network Errors:** HTTP errors include status codes and response body
5. **Cleanup Errors:** Logged but don't fail the build (best effort)

### Error Message Format

Following Go best practices:
- Use `fmt.Errorf("operation: %w", err)` to wrap errors
- Provide context without redundancy
- Avoid words like "failed" or "error" (implicit in error returns)
- Use lowercase messages (Go convention)

Example:
```go
if err != nil {
    return fmt.Errorf("creating instance: %w", err)
}
```

## Security Considerations

1. **Credential Protection:**
   - Credentials are filtered from logs using `packer.LogSecretFilter`
   - Sensitive variables marked in HCL templates

2. **SSH Key Management:**
   - Temporary keys are automatically deleted after build
   - Private keys stored in memory, not on disk (except debug mode)
   - Debug mode key files have 0600 permissions

3. **API Communication:**
   - All API calls use HTTPS
   - Bearer token authentication
   - 30-second timeout on HTTP requests

4. **Resource Cleanup:**
   - Cleanup methods called even on failure
   - Best-effort deletion with error logging

## Testing

### Unit Tests (config_test.go)

Tests configuration validation:
- Required field validation
- Default value assignment
- Timeout duration parsing
- Environment variable reading

Run with: `go test ./builder/crusoe -v`

### Integration Testing

Requires actual Crusoe Cloud credentials. See TESTING.md for details.

## Build and Installation

### Development Build
```bash
make build          # Build binary
make dev            # Build and install to ~/.packer.d/plugins/
make test           # Run unit tests
make fmt            # Format code
make clean          # Remove binary
```

### Production Build
```bash
make install        # Build and install to ~/.packer.d/plugins/
```

## Dependencies

Key dependencies from go.mod:
- `github.com/hashicorp/packer-plugin-sdk` - Packer plugin SDK
- `github.com/hashicorp/hcl/v2` - HCL parsing
- `golang.org/x/crypto` - SSH key generation

## Future Enhancements

Potential improvements:
1. **Mock API Client:** For testing without real API calls
2. **Retry Logic:** Automatic retry on transient API failures
3. **Spot Instances:** Support for spot/preemptible instances
4. **Multiple Disks:** Support for additional disk attachments
5. **Windows Support:** WinRM communicator for Windows images
6. **Image Tags:** Support for tagging custom images
7. **Parallel Builds:** Support for building multiple images concurrently

## Troubleshooting

Common issues and solutions:

### "access_key_id is required"
- Set `CRUSOE_ACCESS_KEY_ID` environment variable or specify in HCL

### "ssh_username must be specified"
- Add `ssh_username = "root"` to your source block (or appropriate user)

### "timeout while waiting for instance"
- Increase `state_timeout` in configuration
- Check Crusoe Cloud status/quotas
- Verify instance type is available in the location

### "no IP address found for instance"
- Check network configuration
- Verify subnet has public IP allocation enabled
- May need to specify `network_id` and `subnet_id`

## Contributing

When contributing to this plugin:
1. Follow Go best practices and conventions
2. Add unit tests for new functionality
3. Update documentation (README, docs/, TESTING.md)
4. Run `make fmt` before committing
5. Ensure `make build` and `make test` pass

## References

- [Packer Plugin Development Guide](https://developer.hashicorp.com/packer/docs/plugins/creation)
- [Crusoe Cloud API Documentation](https://docs.crusoecloud.com/api/)
- [Packer Plugin SDK](https://github.com/hashicorp/packer-plugin-sdk)
- [Go Programming Language](https://golang.org/)

