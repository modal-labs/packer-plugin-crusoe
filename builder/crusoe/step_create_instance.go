package crusoe

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/hashicorp/packer-plugin-sdk/multistep"
	"github.com/hashicorp/packer-plugin-sdk/packer"
)

// isOutOfStockError checks if the operation failed due to an out_of_stock error.
func isOutOfStockError(operation *InstanceOperation) bool {
	if operation == nil {
		return false
	}

	// Try to parse the error code from the operation result
	if operation.Result != nil {
		var errDetail struct {
			Code string `json:"code"`
		}
		if jsonErr := json.Unmarshal(*operation.Result, &errDetail); jsonErr == nil {
			if errDetail.Code == "out_of_stock" {
				return true
			}
		}
	}
	return false
}

type stepCreateInstance struct {
	client *Client
}

func (s *stepCreateInstance) Run(ctx context.Context, state multistep.StateBag) multistep.StepAction {
	c := state.Get("config").(*Config)
	ui := state.Get("ui").(packer.Ui)

	ui.Say("Creating Crusoe instance...")

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
		Location:          c.Location,
		Image:             c.ImageID,
		NetworkInterfaces: networkInterfaces,
	}

	// Check if we're using an ephemeral SSH key pair
	if ephemeralKey, ok := state.GetOk("ephemeral_ssh_key_pair"); ok && ephemeralKey.(bool) {
		// Generate cloud-init script with the ephemeral public key
		if pubKey, ok := state.GetOk("ephemeral_ssh_public_key"); ok {
			sshPublicKey := pubKey.(string)
			cloudInitScript := fmt.Sprintf("#!/bin/bash\nmkdir -p /root/.ssh\nchmod 700 /root/.ssh\necho '%s' >> /root/.ssh/authorized_keys\nchmod 600 /root/.ssh/authorized_keys", sshPublicKey)
			instanceReq.StartupScript = cloudInitScript
			instanceReq.SSHPublicKey = sshPublicKey
			ui.Say("Using ephemeral SSH key with cloud-init script")
		} else {
			err := fmt.Errorf("ephemeral SSH key pair flag set but no public key found in state")
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else if len(c.Comm.SSHPublicKey) > 0 {
		instanceReq.SSHPublicKey = string(c.Comm.SSHPublicKey)
	}

	// If user provided custom userdata, append it to the startup script
	if c.UserData != "" {
		if instanceReq.StartupScript != "" {
			instanceReq.StartupScript = instanceReq.StartupScript + "\n" + c.UserData
		} else {
			instanceReq.StartupScript = c.UserData
		}
	}

	var instanceID string
	var lastErr error
	var triedTypes []string
	instanceTypes := c.InstanceTypes
	if len(instanceTypes) == 0 && c.InstanceType != "" {
		instanceTypes = []string{c.InstanceType}
	}

	// Try each instance type in order, falling back on out_of_stock errors
	for _, instanceType := range instanceTypes {
		instanceReq.Type = instanceType
		triedTypes = append(triedTypes, instanceType)

		if len(instanceTypes) > 1 {
			ui.Say(fmt.Sprintf("Trying instance type: %s", instanceType))
		}

		var operationID string
		instanceID, operationID, lastErr = s.client.CreateInstance(instanceReq)
		if lastErr != nil {
			ui.Error(fmt.Sprintf("creating instance with type %s: %s", instanceType, lastErr))
			state.Put("error", fmt.Errorf("creating instance: %w", lastErr))
			return multistep.ActionHalt
		}

		ui.Say(fmt.Sprintf("Instance %s creation initiated with operation %s", instanceID, operationID))

		ui.Say(fmt.Sprintf("Polling operation %s...", operationID))
		success, operation, err := s.client.PollVMOperationUntilComplete(operationID, c.instanceTimeout)

		if success {
			ui.Say(fmt.Sprintf("Operation %s completed successfully", operationID))
			break
		}

		if isOutOfStockError(operation) {
			ui.Say(fmt.Sprintf("Instance type %s is out of stock", instanceType))
			if len(triedTypes) < len(instanceTypes) {
				ui.Say("Trying next instance type...")
				instanceID = ""
				continue
			}

			lastErr = fmt.Errorf("all attempted instance types are out of stock: tried %s", strings.Join(triedTypes, ", "))
			state.Put("error", lastErr)
			ui.Error(lastErr.Error())
			return multistep.ActionHalt
		}

		// Not an out of stock error.
		if err != nil {
			lastErr = fmt.Errorf("polling operation: %w", err)
		} else if operation != nil {
			if detail := operation.ErrorDetail(); detail != "" {
				lastErr = fmt.Errorf("instance creation operation %s failed (state=%s): %s", operationID, operation.State, detail)
			} else {
				lastErr = fmt.Errorf("instance creation operation %s failed (state=%s): no error details provided by API", operationID, operation.State)
			}
		} else {
			lastErr = fmt.Errorf("instance creation operation %s failed: timed out", operationID)
		}

		state.Put("error", lastErr)
		ui.Error(lastErr.Error())
		return multistep.ActionHalt
	}

	if instanceID == "" {
		lastErr = fmt.Errorf("no instance was created after trying instance types: %s", strings.Join(triedTypes, ", "))
		state.Put("error", lastErr)
		ui.Error(lastErr.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Waiting %ds for instance %s to become active...",
		int(c.instanceTimeout/time.Second), instanceID))

	if err := waitForInstanceState("STATE_RUNNING", instanceID, s.client, c.instanceTimeout); err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	instance, err := s.client.GetInstance(instanceID)
	if err != nil {
		errOut := fmt.Errorf("getting instance: %w", err)
		state.Put("error", errOut)
		ui.Error(errOut.Error())
		return multistep.ActionHalt
	}

	var instanceIP string

	if len(instance.NetworkInterfaces) > 0 && len(instance.NetworkInterfaces[0].IPs) > 0 {
		ips := instance.NetworkInterfaces[0].IPs[0]
		if ips.PublicIPv4 != nil && ips.PublicIPv4.Address != "" {
			instanceIP = ips.PublicIPv4.Address
			ui.Say(fmt.Sprintf("Using public IP: %s", instanceIP))
		} else if ips.PrivateIPv4 != nil && ips.PrivateIPv4.Address != "" {
			instanceIP = ips.PrivateIPv4.Address
			ui.Say(fmt.Sprintf("Using private IP: %s", instanceIP))
		}
	}

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
