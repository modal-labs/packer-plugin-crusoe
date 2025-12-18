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
	defaultStateTimeout = 10 * time.Minute
	defaultAPIEndpoint  = "https://api.crusoecloud.com"
)

// Config provides the config struct
type Config struct {
	common.PackerConfig `mapstructure:",squash"`
	Comm                communicator.Config `mapstructure:",squash"`
	ctx                 interpolate.Context

	// Authentication
	AccessKeyID     string `mapstructure:"access_key_id"`
	SecretAccessKey string `mapstructure:"secret_access_key"`
	APIEndpoint     string `mapstructure:"api_endpoint"`

	// Instance configuration
	Location     string `mapstructure:"location"`
	InstanceType string `mapstructure:"instance_type"`
	ImageID      string `mapstructure:"image_id"`

	// Network configuration
	NetworkID string `mapstructure:"network_id"`
	SubnetID  string `mapstructure:"subnet_id"`

	// SSH configuration
	SSHKeyIDs []string `mapstructure:"ssh_key_ids"`

	// Instance settings
	InstanceName string   `mapstructure:"instance_name"`
	UserData     string   `mapstructure:"userdata"`
	Tags         []string `mapstructure:"tags"`

	// Image settings
	ImageName        string `mapstructure:"image_name"`
	ImageDescription string `mapstructure:"image_description"`

	// Disk settings
	DiskSizeGiB int `mapstructure:"disk_size_gib"`

	// Timeout settings
	RawStateTimeout string `mapstructure:"state_timeout"`

	createTempSSHPair bool

	stateTimeout time.Duration
	interCtx     interpolate.Context
}

// Prepare provides the config prepare functionality
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

	// Validate authentication
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

	if c.APIEndpoint == "" {
		c.APIEndpoint = os.Getenv("CRUSOE_API_ENDPOINT")
		if c.APIEndpoint == "" {
			c.APIEndpoint = defaultAPIEndpoint
		}
	}

	// Validate required fields
	if c.Location == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("location is required"))
	}

	if c.InstanceType == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("instance_type is required"))
	}

	if c.ImageID == "" {
		errs = packer.MultiErrorAppend(errs, errors.New("image_id is required"))
	}

	// Set defaults for optional fields
	if c.ImageName == "" {
		def, err := interpolate.Render("packer-{{timestamp}}", nil)
		if err != nil {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to render image name: %s", err))
		} else {
			c.ImageName = def
		}
	}

	if c.ImageDescription == "" {
		def, err := interpolate.Render("packer-{{timestamp}}", nil)
		if err != nil {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to render image description: %s", err))
		} else {
			c.ImageDescription = def
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

	// Default disk size
	if c.DiskSizeGiB == 0 {
		c.DiskSizeGiB = 50
	}

	// Determine if we need to create a temporary SSH key pair
	if c.Comm.SSHPassword == "" && c.Comm.SSHPrivateKeyFile == "" {
		c.createTempSSHPair = true
	} else {
		c.createTempSSHPair = false
	}

	// Parse state timeout
	if c.RawStateTimeout == "" {
		c.stateTimeout = defaultStateTimeout
	} else {
		if stateTimeout, err := time.ParseDuration(c.RawStateTimeout); err == nil {
			c.stateTimeout = stateTimeout
		} else {
			errs = packer.MultiErrorAppend(errs, fmt.Errorf("unable to parse state timeout: %s", err))
		}
	}

	if es := c.Comm.Prepare(&c.interCtx); len(es) > 0 {
		errs = packer.MultiErrorAppend(errs, es...)
	}

	if errs != nil && len(errs.Errors) > 0 {
		return errs
	}

	packer.LogSecretFilter.Set(c.AccessKeyID, c.SecretAccessKey)

	return nil
}
