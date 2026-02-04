//go:generate packer-sdc mapstructure-to-hcl2 -type Config
package crusoe

import (
	"errors"
	"fmt"
	"os"
	"time"

	common "github.com/hashicorp/packer-plugin-sdk/common"
	"github.com/hashicorp/packer-plugin-sdk/communicator"
	"github.com/hashicorp/packer-plugin-sdk/packer"
	"github.com/hashicorp/packer-plugin-sdk/template/config"
	"github.com/hashicorp/packer-plugin-sdk/template/interpolate"
)

const (
	defaultInstanceTimeout = 20 * time.Minute
	defaultImageTimeout    = 45 * time.Minute
	defaultAPIEndpoint     = "https://api.crusoecloud.com"
	apiCallRetryBackoff    = 10 * time.Second
)

type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`
	ctx                 interpolate.Context

	// Authentication
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	ProjectID       string `mapstructure:"project_id"`
	APIEndpoint     string `mapstructure:"api_endpoint"`

	// Instance configuration
	Location      string   `mapstructure:"location"`
	InstanceType  string   `mapstructure:"instance_type"`
	InstanceTypes []string `mapstructure:"instance_types"` // List of instance types to try in order (fallback on out_of_stock)
	ImageID       string   `mapstructure:"image_id"`

	// Network configuration
	NetworkID string `mapstructure:"network_id"`
	SubnetID  string `mapstructure:"subnet_id"`

	// SSH configuration
	SSHKeyID string `mapstructure:"ssh_key_id"`

	// Instance settings
	InstanceName string   `mapstructure:"instance_name"`
	UserData     string   `mapstructure:"userdata"`
	Tags         []string `mapstructure:"tags"`

	// Image settings
	ImageName        string `mapstructure:"image_name"`
	ImageDescription string `mapstructure:"image_description"`
	DisablePublish   bool   `mapstructure:"disable_publish"`

	// Disk settings
	DiskSizeGiB int `mapstructure:"disk_size_gib"`

	// Timeout settings
	RawInstanceTimeout string `mapstructure:"instance_timeout"`
	RawImageTimeout    string `mapstructure:"image_timeout"`
	RawStateTimeout    string `mapstructure:"state_timeout"` // Deprecated: use instance_timeout

	// Retry settings
	APICallRetries int `mapstructure:"api_call_retries"`

	createTempSSHPair bool

	instanceTimeout time.Duration
	imageTimeout    time.Duration
	interCtx        interpolate.Context
}

func (c *Config) Prepare(raws ...interface{}) error {
	if err := config.Decode(c, &config.DecodeOpts{
		Interpolate:        true,
		InterpolateContext: &c.ctx,
		InterpolateFilter: &interpolate.RenderFilter{
			Exclude: []string{
				"run_command",
			},
		},
	}, raws...); err != nil {
		return err
	}

	var errs *packer.MultiError

	if c.AccessKeyID == "" {
		c.AccessKeyID = os.Getenv("CRUSOE_ACCESS_KEY_ID")
		if c.AccessKeyID == "" {
			errs = packer.MultiErrorAppend(errs, errors.New("access_key_id is required"))
		}
	}

	if c.SecretAccessKey == "" {
		c.SecretAccessKey = os.Getenv("CRUSOE_SECRET_ACCESS_KEY")
		if c.SecretAccessKey == "" {
			errs = packer.MultiErrorAppend(errs, errors.New("secret_access_key is required"))
		}
	}

	if c.ProjectID == "" {
		c.ProjectID = os.Getenv("CRUSOE_PROJECT_ID")
		if c.ProjectID == "" {
			errs = packer.MultiErrorAppend(errs, errors.New("project_id is required"))
		}
	}

	if c.APIEndpoint == "" {
		c.APIEndpoint = os.Getenv("CRUSOE_API_ENDPOINT")
		if c.APIEndpoint == "" {
			c.APIEndpoint = defaultAPIEndpoint
		}
	}

	if c.Location == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("location is required"))
	}

	// If instance_type is set but instance_types is not, use instance_type as the single entry in instance_types.
	// If both are set, instance_types takes precedence
	if len(c.InstanceTypes) == 0 && c.InstanceType != "" {
		c.InstanceTypes = []string{c.InstanceType}
	}

	if len(c.InstanceTypes) == 0 {
		errs = packer.MultiErrorAppend(errs, errors.New("instance_type or instance_types is required"))
	}

	if c.ImageID == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("image_id is required"))
	}

	// Image publish settings (can be disabled to skip creating a custom image)
	if !c.DisablePublish {
		if c.ImageName == "" {
			def, err := interpolate.Render("packer-{{timestamp}}", nil)
			if err != nil {
				errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to render image name: %s", err))
			} else {
				c.ImageName = def
			}
		}

		if len(c.ImageName) >= 50 {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("image_name must be less than 50 characters, got %d characters", len(c.ImageName)))
		}

		if c.ImageDescription == "" {
			def, err := interpolate.Render("packer-{{timestamp}}", nil)
			if err != nil {
				errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to render image description: %s", err))
			} else {
				c.ImageDescription = def
			}
		}
	}

	if c.InstanceName == "" {
		def, err := interpolate.Render("packer-{{timestamp}}", nil)
		if err != nil {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to render instance name: %s", err))
		} else {
			c.InstanceName = def
		}
	}

	if c.DiskSizeGiB == 0 {
		c.DiskSizeGiB = 50
	}

	if c.Comm.SSHPassword == "" && c.Comm.SSHPrivateKeyFile == "" {
		c.createTempSSHPair = true
	} else {
		c.createTempSSHPair = false
	}

	// Parse instance timeout
	if c.RawInstanceTimeout == "" {
		// Fall back to deprecated state_timeout for backwards compatibility
		if c.RawStateTimeout != "" {
			c.RawInstanceTimeout = c.RawStateTimeout
		}
	}

	if c.RawInstanceTimeout == "" {
		c.instanceTimeout = defaultInstanceTimeout
	} else {
		if instanceTimeout, err := time.ParseDuration(c.RawInstanceTimeout); err == nil {
			c.instanceTimeout = instanceTimeout
		} else {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to parse instance timeout: %s", err))
		}
	}

	// Parse image timeout
	if !c.DisablePublish {
		if c.RawImageTimeout == "" {
			c.imageTimeout = defaultImageTimeout
		} else {
			if imageTimeout, err := time.ParseDuration(c.RawImageTimeout); err == nil {
				c.imageTimeout = imageTimeout
			} else {
				errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to parse image timeout: %s", err))
			}
		}
	}

	if es := c.Comm.Prepare(&c.interCtx); len(es) > 0 {
		errs = packer.MultiErrorAppend(errs, es...)
	}

	if c.Comm.SSHPrivateKeyFile != "" && len(c.Comm.SSHPublicKey) == 0 {
		pubKeyPath := c.Comm.SSHPrivateKeyFile + ".pub"

		if pubKeyPath[:2] == "~/" {
			homeDir, err := os.UserHomeDir()
			if err == nil {
				pubKeyPath = homeDir + pubKeyPath[1:]
			}
		}

		if pubKeyData, err := os.ReadFile(pubKeyPath); err == nil {
			c.Comm.SSHPublicKey = pubKeyData
		} else if !c.createTempSSHPair {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("SSH private key file specified but public key file not found at %s", pubKeyPath))
		}
	}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}

	packer.LogSecretFilter.Set(c.AccessKeyID, c.SecretAccessKey)

	return nil
}
