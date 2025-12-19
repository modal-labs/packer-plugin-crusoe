package crusoe

import (
	"fmt"
	"log"

	registryimage 	"github.com/hashicorp/packer-plugin-sdk/packer/registry/image"
)

type Artifact struct {
	ImageID     string
	ImageName   string
	Description string
	Location    string
	client      *Client
	config      *Config
	StateData   map[string]interface{}
}

func (a *Artifact) BuilderId() string {
	return BuilderID
}

func (a *Artifact) Files() []string {
	return nil
}

func (a *Artifact) Id() string {
	return a.ImageID
}

func (a *Artifact) String() string {
	return fmt.Sprintf("Crusoe Custom Image: %s (%s) in %s", a.ImageName, a.ImageID, a.Location)
}

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

func (a *Artifact) Destroy() error {
	log.Printf("Destroying Crusoe Custom Image: %s (%s)", a.ImageID, a.ImageName)
	err := a.client.DeleteCustomImage(a.ImageID)
	return err
}
