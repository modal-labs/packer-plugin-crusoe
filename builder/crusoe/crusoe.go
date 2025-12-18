package crusoe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Client represents a Crusoe API client
type Client struct {
	accessKeyID     string
	secretAccessKey string
	projectID       string
	apiEndpoint     string
	httpClient      *http.Client
}

// NewClient creates a new Crusoe API client
func NewClient(accessKeyID, secretAccessKey, projectID, apiEndpoint string) *Client {
	return &Client{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		projectID:       projectID,
		apiEndpoint:     apiEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with HMAC authentication
func (c *Client) doRequest(method, path string, body interface{}, queryParams url.Values) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	// Remove trailing slashes from path
	path = strings.TrimRight(path, "/")

	fullURL := fmt.Sprintf("%s%s", c.apiEndpoint, path)
	if len(queryParams) > 0 {
		fullURL += "?" + queryParams.Encode()
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	// Add HMAC authentication headers
	if err := c.addAuthHeaders(req); err != nil {
		return nil, fmt.Errorf("add auth headers: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s (URL: %s %s)", resp.StatusCode, string(respBody), method, req.URL.String())
	}

	return respBody, nil
}

// addAuthHeaders adds HMAC-based authentication headers to the request
func (c *Client) addAuthHeaders(req *http.Request) error {
	// Generate RFC3339 timestamp
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Build signature payload
	// Format: http_path\ncanonical_query_params\nhttp_verb\ntimestamp\n
	path := req.URL.Path
	queryParams := c.canonicalizeQueryParams(req.URL.Query())
	method := req.Method
	payload := fmt.Sprintf("%s\n%s\n%s\n%s\n", path, queryParams, method, timestamp)

	// Create HMAC SHA256 signature
	// Per Crusoe API docs, secret key must be base64url-decoded to raw bytes
	secretKeyBytes, err := c.decodeSecretKey()
	if err != nil {
		return fmt.Errorf("decode secret key: %w", err)
	}

	h := hmac.New(sha256.New, secretKeyBytes)
	h.Write([]byte(payload))
	signature := h.Sum(nil)

	// Base64url encode the signature without padding
	signatureB64 := base64.URLEncoding.EncodeToString(signature)
	signatureB64 = strings.TrimRight(signatureB64, "=")

	// Set headers
	req.Header.Set("X-Crusoe-Timestamp", timestamp)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer 1.0:%s:%s", c.accessKeyID, signatureB64))

	return nil
}

// decodeSecretKey decodes the base64url-encoded secret key
func (c *Client) decodeSecretKey() ([]byte, error) {
	// Add padding if necessary
	key := c.secretAccessKey
	if padding := len(key) % 4; padding != 0 {
		key += strings.Repeat("=", 4-padding)
	}

	return base64.URLEncoding.DecodeString(key)
}

// canonicalizeQueryParams canonicalizes query parameters by sorting them lexicographically
func (c *Client) canonicalizeQueryParams(params url.Values) string {
	if len(params) == 0 {
		return ""
	}

	// Sort params by key and format as key=value pairs
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range params[k] {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return strings.Join(pairs, "&")
}

// Instance represents a Crusoe VM instance
type Instance struct {
	ID              string           `json:"id"`
	Name            string           `json:"name"`
	Type            string           `json:"type"`
	Location        string           `json:"location"`
	State           string           `json:"state"`
	PublicIPv4      *Address         `json:"public_ipv4,omitempty"`
	PrivateIPv4     *Address         `json:"private_ipv4,omitempty"`
	NetworkIDs      []string         `json:"network_ids,omitempty"`
	DiskAttachments []DiskAttachment `json:"disk_attachments,omitempty"`
}

// Address represents an IP address
type Address struct {
	Address string `json:"address"`
}

// DiskAttachment represents a disk attachment
type DiskAttachment struct {
	ID   string `json:"id"`
	Mode string `json:"mode"`
}

// CreateInstanceRequest represents the request to create an instance
type CreateInstanceRequest struct {
	Name                string               `json:"name"`
	Type                string               `json:"type"`
	Location            string               `json:"location"`
	Image               string               `json:"image,omitempty"`
	CustomImage         string               `json:"custom_image,omitempty"`
	SSHPublicKey        string               `json:"ssh_public_key,omitempty"`
	StartupScript       string               `json:"startup_script,omitempty"`
	NetworkInterfaces   []NetworkInterface   `json:"network_interfaces,omitempty"`
	HostChannelAdapters []HostChannelAdapter `json:"host_channel_adapters,omitempty"`
}

// NetworkInterface represents a network interface configuration
type NetworkInterface struct {
	IPs []NetworkIP `json:"ips,omitempty"`
}

// NetworkIP represents an IP configuration
type NetworkIP struct {
	PublicIPv4 *PublicIPv4Config `json:"public_ipv4,omitempty"`
}

// PublicIPv4Config represents public IPv4 configuration
type PublicIPv4Config struct {
	Type string `json:"type"` // "static" or "dynamic"
}

// HostChannelAdapter represents an InfiniBand partition configuration
type HostChannelAdapter struct {
	IBPartitionID string `json:"ib_partition_id"`
}

// DiskCreateRequest represents a disk creation request (kept for compatibility)
type DiskCreateRequest struct {
	SizeGiB int    `json:"size_gib"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
}

// CreateInstanceResponse represents the response from creating an instance
type CreateInstanceResponse struct {
	Operation InstanceOperation `json:"operation"`
}

// InstanceOperation represents an asynchronous operation
type InstanceOperation struct {
	OperationID string            `json:"operation_id"`
	State       string            `json:"state"`
	Metadata    OperationMetadata `json:"metadata"`
	Result      *json.RawMessage  `json:"result,omitempty"`
	StartedAt   string            `json:"started_at"`
	CompletedAt string            `json:"completed_at,omitempty"`
}

// OperationMetadata contains metadata about the operation
type OperationMetadata struct {
	OperationName string `json:"operation_name"`
	ID            string `json:"id"` // The VM ID
	Type          string `json:"type"`
}

// OperationStatus represents the status of an async operation
type OperationStatus int

const (
	OperationStatusPending OperationStatus = iota
	OperationStatusSucceeded
	OperationStatusFailed
)

// GetVMOperationStatus queries the Crusoe API for the status of a VM operation
func (c *Client) GetVMOperationStatus(operationID string) (OperationStatus, *InstanceOperation, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/operations/%s", c.projectID, operationID)

	respBody, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return OperationStatusFailed, nil, err
	}

	var operation InstanceOperation
	if err := json.Unmarshal(respBody, &operation); err != nil {
		return OperationStatusFailed, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	state := strings.ToUpper(operation.State)
	switch state {
	case "SUCCEEDED":
		return OperationStatusSucceeded, &operation, nil
	case "FAILED":
		return OperationStatusFailed, &operation, nil
	case "PENDING", "IN_PROGRESS":
		return OperationStatusPending, &operation, nil
	default:
		return OperationStatusFailed, &operation, fmt.Errorf("unknown operation state: %s", state)
	}
}

// PollVMOperationUntilComplete polls a VM operation until it completes (SUCCEEDED or FAILED)
func (c *Client) PollVMOperationUntilComplete(operationID string, timeout time.Duration) (bool, *InstanceOperation, error) {
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		status, operation, err := c.GetVMOperationStatus(operationID)
		if err != nil {
			return false, nil, err
		}

		switch status {
		case OperationStatusSucceeded:
			return true, operation, nil
		case OperationStatusFailed:
			return false, operation, nil
		case OperationStatusPending:
			// Continue polling
			time.Sleep(2 * time.Second)
		}
	}

	return false, nil, fmt.Errorf("operation timed out after %v", timeout)
}

// GetImageOperationStatus queries the Crusoe API for the status of a custom image operation
func (c *Client) GetImageOperationStatus(operationID string) (OperationStatus, *InstanceOperation, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/images/operations/%s", c.projectID, operationID)

	respBody, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return OperationStatusFailed, nil, err
	}

	var operation InstanceOperation
	if err := json.Unmarshal(respBody, &operation); err != nil {
		return OperationStatusFailed, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	state := strings.ToUpper(operation.State)
	switch state {
	case "SUCCEEDED":
		return OperationStatusSucceeded, &operation, nil
	case "FAILED":
		return OperationStatusFailed, &operation, nil
	case "PENDING", "IN_PROGRESS":
		return OperationStatusPending, &operation, nil
	default:
		return OperationStatusFailed, &operation, fmt.Errorf("unknown operation state: %s", state)
	}
}

// PollImageOperationUntilComplete polls a custom image operation until it completes (SUCCEEDED or FAILED)
func (c *Client) PollImageOperationUntilComplete(operationID string, timeout time.Duration) (bool, *InstanceOperation, error) {
	startTime := time.Now()

	for time.Since(startTime) < timeout {
		status, operation, err := c.GetImageOperationStatus(operationID)
		if err != nil {
			return false, nil, err
		}

		switch status {
		case OperationStatusSucceeded:
			return true, operation, nil
		case OperationStatusFailed:
			return false, operation, nil
		case OperationStatusPending:
			// Continue polling
			time.Sleep(2 * time.Second)
		}
	}

	return false, nil, fmt.Errorf("operation timed out after %v", timeout)
}

// CreateInstance creates a new VM instance
func (c *Client) CreateInstance(req *CreateInstanceRequest) (*Instance, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances", c.projectID)

	respBody, err := c.doRequest("POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var resp CreateInstanceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	// Validate that we got an instance ID and operation ID
	if resp.Operation.Metadata.ID == "" {
		return nil, fmt.Errorf("API did not return an instance ID. Response: %s", string(respBody))
	}
	if resp.Operation.OperationID == "" {
		return nil, fmt.Errorf("API did not return an operation ID. Response: %s", string(respBody))
	}

	// Return an Instance with the ID and operation ID
	instance := &Instance{
		ID:       resp.Operation.Metadata.ID,
		Name:     req.Name,
		Type:     req.Type,
		Location: req.Location,
		State:    "PROVISIONING", // Initial state
	}

	// Store the operation ID in the instance for polling
	// We'll add this field to the Instance struct
	state := resp.Operation.State
	_ = state // Use the operation state if needed

	return instance, nil
}

// GetInstance retrieves an instance by ID
func (c *Client) GetInstance(instanceID string) (*Instance, error) {
	if instanceID == "" {
		return nil, fmt.Errorf("instance ID cannot be empty")
	}

	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/%s", c.projectID, instanceID)

	respBody, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var instance Instance
	if err := json.Unmarshal(respBody, &instance); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &instance, nil
}

// UpdateInstanceRequest represents the request to update an instance
type UpdateInstanceRequest struct {
	State string `json:"state,omitempty"`
}

// UpdateInstance updates an instance (e.g., to shut it down)
func (c *Client) UpdateInstance(instanceID string, req *UpdateInstanceRequest) error {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/%s", c.projectID, instanceID)

	_, err := c.doRequest("PATCH", path, req, nil)
	return err
}

// DeleteInstance deletes an instance
func (c *Client) DeleteInstance(instanceID string) error {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/%s", c.projectID, instanceID)

	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}

// CustomImage represents a custom image
type CustomImage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	State       string `json:"state"`
	Location    string `json:"location"`
}

// CreateCustomImageRequest represents the request to create a custom image
type CreateCustomImageRequest struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location"`
	SourceDisk  string `json:"source_disk"`
}

// CreateCustomImageResponse represents the response from creating a custom image
type CreateCustomImageResponse struct {
	Image CustomImage `json:"image"`
}

// CreateCustomImage creates a custom image from a disk
func (c *Client) CreateCustomImage(req *CreateCustomImageRequest) (*CustomImage, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/custom-images", c.projectID)

	respBody, err := c.doRequest("POST", path, req, nil)
	if err != nil {
		return nil, err
	}

	var resp CreateCustomImageResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp.Image, nil
}

// GetCustomImage retrieves a custom image by ID
func (c *Client) GetCustomImage(imageID string) (*CustomImage, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/custom-images/%s", c.projectID, imageID)

	respBody, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return nil, err
	}

	var image CustomImage
	if err := json.Unmarshal(respBody, &image); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &image, nil
}

// DeleteCustomImage deletes a custom image
func (c *Client) DeleteCustomImage(imageID string) error {
	path := fmt.Sprintf("/v1alpha5/compute/custom-images/%s", imageID)

	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}

// SSHKey represents an SSH key
type SSHKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// CreateSSHKeyRequest represents the request to create an SSH key
type CreateSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

// CreateSSHKeyResponse represents the response from creating an SSH key
type CreateSSHKeyResponse struct {
	SSHKey SSHKey `json:"ssh_key"`
}

// CreateSSHKey creates an SSH key
func (c *Client) CreateSSHKey(req *CreateSSHKeyRequest) (*SSHKey, error) {
	respBody, err := c.doRequest("POST", "/users/ssh-keys", req, nil)
	if err != nil {
		return nil, err
	}

	var resp CreateSSHKeyResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp.SSHKey, nil
}

// DeleteSSHKey deletes an SSH key
func (c *Client) DeleteSSHKey(keyID string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/users/ssh-keys/%s", keyID), nil, nil)
	return err
}
