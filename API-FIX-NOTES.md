# API Fix Notes

## Issue: Invalid Action Error When Stopping Instance

### Problem
When Packer tried to stop an instance before creating an image, it received:
```
API request failed with status 400: {"code":"bad_request","message":"bad request, check request parameters: invalid action"}
```

### Root Cause
The `UpdateInstanceRequest` was sending:
```json
{
  "state": "stopped"
}
```

But according to the [Crusoe API documentation](https://docs.crusoecloud.com/api/#tag/VMs/operation/updateInstance), the PATCH request should use:
```json
{
  "action": "STOP"
}
```

### Fix Applied

#### 1. Updated `UpdateInstanceRequest` struct in `crusoe.go`:
```go
type UpdateInstanceRequest struct {
    Action string `json:"action"`  // Changed from: State string `json:"state,omitempty"`
}
```

#### 2. Updated `step_shutdown.go` to send correct action:
```go
updateReq := &UpdateInstanceRequest{
    Action: "STOP",  // Changed from: State: "stopped"
}
```

#### 3. Updated state checking to use correct state name:
```go
waitForInstanceState("STATE_STOPPED", ...)  // Changed from: "stopped"
```

### Valid Actions
According to the Crusoe API, valid actions for the PATCH `/projects/{project_id}/compute/vms/instances/{vm_id}` endpoint are:
- `START` - Start a stopped instance
- `STOP` - Stop a running instance
- `RESTART` - Restart an instance

**Note:** Actions must be in UPPERCASE.

### Instance States
The API returns instance states in uppercase with `STATE_` prefix:
- `STATE_RUNNING`
- `STATE_STOPPED`
- `STATE_PROVISIONING`
- etc.

### Testing
After this fix, the stop operation should succeed:
```bash
==> crusoe.region-eu-iceland1-a: Using API to stop instance...
==> crusoe.region-eu-iceland1-a: Waiting for instance to stop...
==> crusoe.region-eu-iceland1-a: Creating custom image...
```

