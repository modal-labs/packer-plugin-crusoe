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

	// Get SSH public key - prefer temp key if available
	var sshPublicKey string
	if pubKey, ok := state.GetOk("temp_ssh_public_key"); ok {
		sshPublicKey = pubKey.(string)
	} else if c.Comm.SSHPublicKey != nil && len(c.Comm.SSHPublicKey) > 0 {
		// Use the public key from the communicator config
		sshPublicKey = string(c.Comm.SSHPublicKey)
	} else {
		// If no public key is available, return an error
		err := fmt.Errorf("SSH public key is required. Please specify ssh_private_key_file (with corresponding .pub file) or let Packer create a temporary key pair")
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Configure network interface with static public IP
	networkInterfaces := []NetworkInterface{
		{
			IPs: []NetworkIP{
				{
					PublicIPv4: &PublicIPv4Config{
						Type: "static",
					},
				},
			},
		},
	}

	instanceReq := &CreateInstanceRequest{
		Name:              c.InstanceName,
		Type:              c.InstanceType,
		Location:          c.Location,
		Image:             c.ImageID,
		SSHPublicKey:      sshPublicKey,
		StartupScript:     c.UserData,
		NetworkInterfaces: networkInterfaces,
	}

	instanceID, operationID, err := s.client.CreateInstance(instanceReq)
	if err != nil {
		errOut := fmt.Errorf("creating instance: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Instance %s creation initiated with operation %s", instanceID, operationID))

	// First, poll the operation endpoint until the operation completes
	ui.Say(fmt.Sprintf("Polling operation %s...", operationID))
	success, operation, err := s.client.PollVMOperationUntilComplete(operationID, c.stateTimeout)
	if err != nil {
		errOut := fmt.Errorf("polling operation: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}
	if !success {
		errOut := fmt.Errorf("operation %s failed: state=%s", operationID, operation.State)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Operation %s completed successfully", operationID))

	// Now poll the instance endpoint until the instance is running
	ui.Say(fmt.Sprintf("Waiting %ds for instance %s to become active...",
		int(c.stateTimeout/time.Second), instanceID))

	if err = waitForInstanceState("STATE_RUNNING", instanceID, s.client, c.stateTimeout); err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	// Get the updated instance with IP address
	instance, err := s.client.GetInstance(instanceID)
	if err != nil {
		errOut := fmt.Errorf("getting instance: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	// Determine IP address to use for SSH
	var instanceIP string

	// Parse IP addresses from network interfaces
	if len(instance.NetworkInterfaces) > 0 && len(instance.NetworkInterfaces[0].IPs) > 0 {
		ips := instance.NetworkInterfaces[0].IPs[0]
		// Prefer public IP if available
		if ips.PublicIPv4 != nil && ips.PublicIPv4.Address != "" {
			instanceIP = ips.PublicIPv4.Address
			ui.Say(fmt.Sprintf("Using public IP: %s", instanceIP))
		} else if ips.PrivateIPv4 != nil && ips.PrivateIPv4.Address != "" {
			instanceIP = ips.PrivateIPv4.Address
			ui.Say(fmt.Sprintf("Using private IP: %s", instanceIP))
		}
	}

	// Fallback to legacy fields if new structure is not available
	if instanceIP == "" {
		if instance.PublicIPv4 != nil && instance.PublicIPv4.Address != "" {
			instanceIP = instance.PublicIPv4.Address
		} else if instance.PrivateIPv4 != nil && instance.PrivateIPv4.Address != "" {
			instanceIP = instance.PrivateIPv4.Address
		}
	}

	if instanceIP == "" {
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
