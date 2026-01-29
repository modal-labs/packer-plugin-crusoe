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

const preImageCreationDelay = 30

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

	// Ensure that the instance is STATE_SHUTOFF
	if err := waitForInstanceState("STATE_SHUTOFF", instance.ID, s.client, c.instanceTimeout); err != nil {
		errOut := fmt.Sprintf("error ensuring instance is shut off: %s", err.Error())
		state.Put("error", errOut)
		ui.Error(errOut)
		return multistep.ActionHalt
	}
	time.Sleep(preImageCreationDelay * time.Second)

	imageReq := &CreateCustomImageRequest{
		DiskID:      diskID,
		Name:        c.ImageName,
		Description: c.ImageDescription,
	}

	retries := c.APICallRetries
	if retries < 0 {
		retries = 0
	}

	attempts := retries + 1
	var operationID string
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			ui.Say(fmt.Sprintf("Retrying create image snapshot API call (attempt %d/%d)...", attempt+1, attempts))
			time.Sleep(apiCallRetryBackoff)
		}

		operationID, err = s.client.CreateCustomImage(imageReq)
		if err != nil {
			ui.Say(fmt.Sprintf("Create image snapshot API call failed (attempt %d/%d): %s", attempt+1, attempts, err))
			continue // Retry.
		}

		ui.Say(fmt.Sprintf("Custom image creation started (operation: %s)", operationID))
		ui.Say("Polling for image creation operation to complete...")

		success, operation, err := s.client.PollImageOperationUntilComplete(operationID, c.imageTimeout)
		if err != nil {
			errOut := fmt.Errorf("polling image operation: %w", err)
			state.Put("error", errOut)
			ui.Error(errOut.Error())
			continue // Retry.
		}

		if !success {
			if operation != nil {
				var errOut error
				if detail := operation.ErrorDetail(); detail != "" {
					errOut = fmt.Errorf("image creation operation failed (state=%s): %s", operation.State, detail)
				} else {
					errOut = fmt.Errorf("image creation operation failed (state=%s): no error details provided by API", operation.State)
				}
				state.Put("error", errOut)
				ui.Error(errOut.Error())
			} else {
				errOut := fmt.Errorf("image creation operation timed out")
				state.Put("error", errOut)
				ui.Error(errOut.Error())
			}
			continue // Retry.
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

	// Halt if all retries failed.
	errOut := fmt.Errorf("Creating custom image: %w failed after %d attempts", err, attempts)
	state.Put("error", errOut)
	ui.Error(errOut.Error())
	return multistep.ActionHalt

}

func (s *stepCreateImage) Cleanup(state multistep.StateBag) {
}
