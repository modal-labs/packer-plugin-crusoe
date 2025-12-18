package crusoe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateImage struct {
	client *Client
}

// Run provides the step create image run functionality
func (s *stepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	instance := state.Get("instance").(*Instance)

	ui.Say("Creating custom image from instance disk...")

	// Get the disk ID from the instance
	if len(instance.DiskAttachments) == 0 {
		errOut := fmt.Errorf("no disk attachments found on instance %s", instance.ID)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	diskID := instance.DiskAttachments[0].ID

	imageReq := &CreateCustomImageRequest{
		Name:        c.ImageName,
		Description: c.ImageDescription,
		Location:    c.Location,
		SourceDisk:  diskID,
	}

	image, err := s.client.CreateCustomImage(imageReq)
	if err != nil {
		errOut := fmt.Errorf("creating custom image: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Waiting %ds for image %s to be ready...",
		int(c.stateTimeout/time.Second), image.ID))

	err = waitForImageState("available", image.ID, s.client, c.stateTimeout)
	if err != nil {
		errOut := fmt.Errorf("waiting for image: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Custom image %s created successfully", image.ID))

	state.Put("image", image)
	return multistep.ActionContinue
}

// Cleanup provides the step create image cleanup functionality
func (s *stepCreateImage) Cleanup(state multistep.StateBag) {
}
