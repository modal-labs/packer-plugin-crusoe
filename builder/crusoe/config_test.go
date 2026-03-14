package crusoe

import (
	"testing"
)

func TestConfigPrepare_Required(t *testing.T) {
	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "missing access_key_id",
			config: map[string]interface{}{
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
			},
			wantErr: true,
		},
		{
			name: "missing secret_access_key",
			config: map[string]interface{}{
				"access_key_id": "test-access",
				"project_id":    "test-project",
				"location":      "us-northcentral1-a",
				"instance_type": "a40.1x",
				"image_id":      "ubuntu22.04:latest",
			},
			wantErr: true,
		},
		{
			name: "missing project_id",
			config: map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
			},
			wantErr: true,
		},
		{
			name: "missing location",
			config: map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
			},
			wantErr: true,
		},
		{
			name: "missing instance_type",
			config: map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"image_id":          "ubuntu22.04:latest",
			},
			wantErr: true,
		},
		{
			name: "missing image_id",
			config: map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
			},
			wantErr: true,
		},
		{
			name: "valid minimal config",
			config: map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
				"ssh_username":      "root",
			},
			wantErr: false,
		},
		{
			name: "valid config with all optional fields",
			config: map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
				"network_id":        "net-123",
				"subnet_id":         "subnet-456",
				"instance_name":     "test-instance",
				"image_name":        "test-image",
				"image_description": "Test image description",
				"disk_size_gib":     100,
				"state_timeout":     "15m",
				"ssh_key_id":        "key-1",
				"tags":              []string{"tag1", "tag2"},
				"ssh_username":      "root",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Config
			err := c.Prepare(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigPrepare_Defaults(t *testing.T) {
	config := map[string]interface{}{
		"access_key_id":     "test-access",
		"secret_access_key": "test-secret",
		"project_id":        "test-project",
		"location":          "us-northcentral1-a",
		"instance_type":     "a40.1x",
		"image_id":          "ubuntu22.04:latest",
		"ssh_username":      "root",
	}

	var c Config
	err := c.Prepare(config)
	if err != nil {
		t.Fatalf("Config.Prepare() unexpected error = %v", err)
	}

	// Check default values
	if c.APIEndpoint != defaultAPIEndpoint {
		t.Errorf("APIEndpoint = %v, want %v", c.APIEndpoint, defaultAPIEndpoint)
	}

	if c.DiskSizeGiB != 50 {
		t.Errorf("DiskSizeGiB = %v, want 50", c.DiskSizeGiB)
	}

	if c.instanceTimeout != defaultInstanceTimeout {
		t.Errorf("instanceTimeout = %v, want %v", c.instanceTimeout, defaultInstanceTimeout)
	}

	if c.imageTimeout != defaultImageTimeout {
		t.Errorf("imageTimeout = %v, want %v", c.imageTimeout, defaultImageTimeout)
	}

	if c.ImageName == "" {
		t.Error("ImageName should have a default value")
	}

	if c.ImageDescription == "" {
		t.Error("ImageDescription should have a default value")
	}

	if c.InstanceName == "" {
		t.Error("InstanceName should have a default value")
	}
}

func TestConfigPrepare_Timeouts(t *testing.T) {
	tests := []struct {
		name            string
		instanceTimeout string
		imageTimeout    string
		stateTimeout    string // For backwards compatibility testing
		wantErr         bool
	}{
		{
			name:            "valid instance timeout 15m",
			instanceTimeout: "15m",
			wantErr:         false,
		},
		{
			name:         "valid image timeout 1h",
			imageTimeout: "1h",
			wantErr:      false,
		},
		{
			name:            "valid both timeouts",
			instanceTimeout: "20m",
			imageTimeout:    "45m",
			wantErr:         false,
		},
		{
			name:            "invalid instance timeout",
			instanceTimeout: "invalid",
			wantErr:         true,
		},
		{
			name:         "invalid image timeout",
			imageTimeout: "invalid",
			wantErr:      true,
		},
		{
			name:         "backwards compatible state_timeout",
			stateTimeout: "15m",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
				"ssh_username":      "root",
			}

			if tt.instanceTimeout != "" {
				config["instance_timeout"] = tt.instanceTimeout
			}
			if tt.imageTimeout != "" {
				config["image_timeout"] = tt.imageTimeout
			}
			if tt.stateTimeout != "" {
				config["state_timeout"] = tt.stateTimeout
			}

			var c Config
			err := c.Prepare(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConfigPrepare_ImageNameLength(t *testing.T) {
	tests := []struct {
		name      string
		imageName string
		wantErr   bool
	}{
		{
			name:      "valid image name",
			imageName: "my-custom-image",
			wantErr:   false,
		},
		{
			name:      "image name with 49 characters",
			imageName: "1234567890123456789012345678901234567890123456789",
			wantErr:   false,
		},
		{
			name:      "image name with 50 characters",
			imageName: "12345678901234567890123456789012345678901234567890",
			wantErr:   true,
		},
		{
			name:      "image name with 51 characters",
			imageName: "123456789012345678901234567890123456789012345678901",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := map[string]interface{}{
				"access_key_id":     "test-access",
				"secret_access_key": "test-secret",
				"project_id":        "test-project",
				"location":          "us-northcentral1-a",
				"instance_type":     "a40.1x",
				"image_id":          "ubuntu22.04:latest",
				"image_name":        tt.imageName,
				"ssh_username":      "root",
			}

			var c Config
			err := c.Prepare(config)
			if (err != nil) != tt.wantErr {
				t.Errorf("Config.Prepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
