package crusoe

import (
	"context"
	"fmt"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateImage struct {
	client *Client
}

func (s *stepCreateImage) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)
	instance := state.Get("instance").(*Instance)

	ui.Say("Creating custom image from instance disk...")

	if len(instance.Disks) == 0 {
		errOut := fmt.Errorf("no disk attachments found on instance %s", instance.ID)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	diskID := instance.Disks[0].ID
	ui.Say(fmt.Sprintf("Found disk ID: %s", diskID))

	imageReq := &CreateCustomImageRequest{
		DiskID:      diskID,
		Name:        c.ImageName,
		Description: c.ImageDescription,
	}

	operationID, err := s.client.CreateCustomImage(imageReq)
	if err != nil {
		errOut := fmt.Errorf("creating custom image: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Custom image creation started (operation: %s)", operationID))
	ui.Say("Polling for image creation operation to complete...")

	success, operation, err := s.client.PollImageOperationUntilComplete(operationID, c.stateTimeout)
	if err != nil {
		errOut := fmt.Errorf("polling image operation: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	if !success {
		if operation != nil {
			errOut := fmt.Errorf("image creation operation failed: %s", operation.State)
			state.Put("error", errOut)
			ui.Error(errOut.Error())
		} else {
			errOut := fmt.Errorf("image creation operation timed out")
			state.Put("error", errOut)
			ui.Error(errOut.Error())
		}
		return multistep.ActionHalt
	}

	imageID := operation.Metadata.ID
	ui.Say(fmt.Sprintf("Custom image created successfully (ID: %s)", imageID))

	image := &CustomImage{
		ID:          imageID,
		Name:        c.ImageName,
		Description: c.ImageDescription,
		Location:    c.Location,
	}

	state.Put("image", image)
	return multistep.ActionContinue
}

func (s *stepCreateImage) Cleanup(state multistep.StateBag) {
}
