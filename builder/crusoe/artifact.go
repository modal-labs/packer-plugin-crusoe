package crusoe

import (
	"fmt"
	"log"

	registryimage "github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
)

// Artifact provides the artifact struct
type Artifact struct {
	// The ID of the image
	ImageID string

	// The name of the image
	ImageName string

	// The description of the image
	Description string

	// The location of the image
	Location string

	// The client for making changes
	client *Client

	// config definition from the builder
	config *Config

	// State data used by HCP container registry
	StateData map[string]interface{}
}

// BuilderId provides the builder ID
func (a *Artifact) BuilderId() string {
	return BuilderID
}

// Files provides nil
func (a *Artifact) Files() []string {
	return nil
}

// Id provides the image ID
func (a *Artifact) Id() string {
	return a.ImageID
}

// String provides the image description and ID in a string
func (a *Artifact) String() string {
	return fmt.Sprintf("Crusoe Custom Image: %s (%s) in %s", a.ImageName, a.ImageID, a.Location)
}

// State provides the artifact state
func (a *Artifact) State(name string) interface{} {
	if name == registryimage.ArtifactStateURI {
		img, err := registryimage.FromArtifact(a,
			registryimage.WithID(a.ImageID),
			registryimage.WithProvider("Crusoe"),
			registryimage.WithRegion(a.config.Location),
		)

		if err != nil {
			log.Printf("[DEBUG] error encountered when creating a registry image %v", err)
			return nil
		}
		return img
	}
	return a.StateData[name]
}

// Destroy destroys the artifact image
func (a *Artifact) Destroy() error {
	log.Printf("Destroying Crusoe Custom Image: %s (%s)", a.ImageID, a.ImageName)
	err := a.client.DeleteCustomImage(a.ImageID)
	return err
}
