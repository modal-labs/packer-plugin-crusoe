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

	ui.Say("Performing graceful shutdown...")
	time.Sleep(ShutdownDelaySec * time.Second)

	comm := state.Get("communicator").(packer.Communicator)

	cmd := &packer.RemoteCmd{
		Command: "sudo shutdown -h now",
	}

	if err := comm.Start(ctx, cmd); err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}
	cmd.Wait()

	if cmd.ExitStatus() == packer.CmdDisconnect {
		ui.Say("Instance successfully shutdown via SSH")
		time.Sleep(ShutdownDelaySec * time.Second)
	} else {
		ui.Say("Using API to stop instance...")
	}

	updateReq := &UpdateInstanceRequest{
		Action: "STOP",
	}
	if err := s.client.UpdateInstance(instance.ID, updateReq); err != nil {
		errOut := fmt.Errorf("stopping instance: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Waiting for instance %s to stop...", instance.ID))
	if err := waitForInstanceState("STATE_SHUTOFF", instance.ID, s.client, c.stateTimeout); err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say("Instance successfully stopped")
	return multistep.ActionContinue
}

func (s *stepShutdown) Cleanup(state multistep.StateBag) {
}
