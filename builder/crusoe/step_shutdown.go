package crusoe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

const ShutdownDelaySec = 10

type stepShutdown struct {
	client *Client
}

func (s *stepShutdown) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	c := state.Get("config").(*Config)
	instance := state.Get("instance").(*Instance)

	ui.Say("Using API to stop instance...")

	updateReq := &UpdateInstanceRequest{
		Action: "STOP",
	}

	attempts := max(0, c.APICallRetries+1)
	var err error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			ui.Say(fmt.Sprintf("Retrying power off instance API call (attempt %d/%d)...", attempt+1, attempts))
			time.Sleep(apiCallRetryBackoff)
		}

		err = s.client.UpdateInstance(instance.ID, updateReq)
		if err != nil {
			ui.Say(fmt.Sprintf("Power off instance API call failed (attempt %d/%d): %s", attempt+1, attempts, err))
			continue // Retry.
		}

		ui.Say(fmt.Sprintf("Waiting for instance %s to stop...", instance.ID))
		if waitErr := waitForInstanceState("STATE_SHUTOFF", instance.ID, s.client, c.instanceTimeout); waitErr != nil {
			err = waitErr
			ui.Error(err.Error())
			continue // Retry.
		}

		ui.Say("Instance successfully stopped")
		return multistep.ActionContinue
	}

	// All attempts failed.
	errOut := fmt.Errorf("stopping instance: %w", err)
	state.Put("error", errOut)
	ui.Error(errOut.Error())
	return multistep.ActionHalt
}

func (s *stepShutdown) Cleanup(state multistep.StateBag) {
}
