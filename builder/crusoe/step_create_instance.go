package crusoe

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

type stepCreateInstance struct {
	client *Client
}

// Run provides the step create instance run functionality
func (s *stepCreateInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Creating Crusoe instance...")

	sshKeys := c.SSHKeyIDs
	key, keyOK := state.GetOk("temp_ssh_key_id")
	if keyOK {
		sshKeys = append(sshKeys, key.(string))
	}

	// Build disk request
	disks := []DiskCreateRequest{
		{
			SizeGiB: c.DiskSizeGiB,
			Type:    "persistent-ssd",
			Mode:    "read-write",
		},
	}

	instanceReq := &CreateInstanceRequest{
		Name:      c.InstanceName,
		Type:      c.InstanceType,
		Location:  c.Location,
		Image:     c.ImageID,
		SSHKeys:   sshKeys,
		UserData:  c.UserData,
		Tags:      c.Tags,
		NetworkID: c.NetworkID,
		SubnetID:  c.SubnetID,
		Disks:     disks,
	}

	instance, err := s.client.CreateInstance(instanceReq)
	if err != nil {
		errOut := fmt.Errorf("creating instance: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	// Wait until instance is running
	ui.Say(fmt.Sprintf("Waiting %ds for instance %s to become active...",
		int(c.stateTimeout/time.Second), instance.ID))

	if err = waitForInstanceState("running", instance.ID, s.client, c.stateTimeout); err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Get the updated instance with IP address
	if instance, err = s.client.GetInstance(instance.ID); err != nil {
		errOut := fmt.Errorf("getting instance: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	// Determine IP address to use for SSH
	var instanceIP string
	if instance.PublicIPv4 != nil && instance.PublicIPv4.Address != "" {
		instanceIP = instance.PublicIPv4.Address
	} else if instance.PrivateIPv4 != nil && instance.PrivateIPv4.Address != "" {
		instanceIP = instance.PrivateIPv4.Address
	} else {
		errOut := fmt.Errorf("no IP address found for instance %s", instance.ID)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Instance %s is running at %s", instance.ID, instanceIP))

	state.Put("instance", instance)
	state.Put("instance_ip", instanceIP)
	state.Put("instance_id", instance.ID)

	return multistep.ActionContinue
}

// Cleanup provides the step create instance cleanup functionality
func (s *stepCreateInstance) Cleanup(state multistep.StateBag) {
	instance, ok := state.GetOk("instance")
	if !ok {
		return
	}

	ui := state.Get("ui").(packer.Ui)
	inst := instance.(*Instance)

	ui.Say("Destroying instance " + inst.ID)
	if err := s.client.DeleteInstance(inst.ID); err != nil {
		state.Put("error", err)
	}
}
