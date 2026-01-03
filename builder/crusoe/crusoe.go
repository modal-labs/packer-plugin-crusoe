package crusoe

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type Client struct {
	accessKeyID     string
	secretAccessKey string
	projectID       string
	apiEndpoint     string
	httpClient      *http.Client
}

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

func (c *Client) doRequest(method, path string, body interface{}, queryParams url.Values) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		log.Printf("[DEBUG] request body: %s", string(jsonData))
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	path = strings.TrimRight(path, "/")

	fullURL := fmt.Sprintf("%s%s", c.apiEndpoint, path)
	if len(queryParams) > 0 {
		fullURL += "?" + queryParams.Encode()
	}

	req, err := http.NewRequest(method, fullURL, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if err := c.addAuthHeaders(req); err != nil {
		return nil, fmt.Errorf("add auth headers: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	log.Printf("[DEBUG] API Request: %s %s", method, req.URL.String())

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	log.Printf("[DEBUG] API Response: status=%d, body=%s", resp.StatusCode, string(respBody))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("API request failed with status %d: %s (URL: %s %s)", resp.StatusCode, string(respBody), method, req.URL.String())
	}

	return respBody, nil
}

func (c *Client) addAuthHeaders(req *http.Request) error {
	timestamp := time.Now().UTC().Format(time.RFC3339)

	// Format: http_path\ncanonical_query_params\nhttp_verb\ntimestamp\n
	path := req.URL.Path
	queryParams := c.canonicalizeQueryParams(req.URL.Query())
	method := req.Method
	payload := fmt.Sprintf("%s\n%s\n%s\n%s\n", path, queryParams, method, timestamp)

	// Per Crusoe API docs, secret key must be base64url-decoded to raw bytes
	secretKeyBytes, err := c.decodeSecretKey()
	if err != nil {
		return fmt.Errorf("decode secret key: %w", err)
	}

	h := hmac.New(sha256.New, secretKeyBytes)
	h.Write([]byte(payload))
	signature := h.Sum(nil)

	signatureB64 := base64.URLEncoding.EncodeToString(signature)
	signatureB64 = strings.TrimRight(signatureB64, "=") // Remove padding

	req.Header.Set("X-Crusoe-Timestamp", timestamp)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer 1.0:%s:%s", c.accessKeyID, signatureB64))

	return nil
}

func (c *Client) decodeSecretKey() ([]byte, error) {
	key := c.secretAccessKey
	if padding := len(key) % 4; padding != 0 {
		key += strings.Repeat("=", 4-padding)
	}

	return base64.URLEncoding.DecodeString(key)
}

func (c *Client) canonicalizeQueryParams(params url.Values) string {
	if len(params) == 0 {
		return ""
	}

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

type Instance struct {
	ID                string                     `json:"id"`
	Name              string                     `json:"name"`
	Type              string                     `json:"type"`
	Location          string                     `json:"location"`
	State             string                     `json:"state"`
	NetworkInterfaces []InstanceNetworkInterface `json:"network_interfaces,omitempty"`
	Disks             []DiskAttachment           `json:"disks,omitempty"`

	// Deprecated: Use NetworkInterfaces instead
	PublicIPv4  *Address `json:"public_ipv4,omitempty"`
	PrivateIPv4 *Address `json:"private_ipv4,omitempty"`
	NetworkIDs  []string `json:"network_ids,omitempty"`
}

type InstanceNetworkInterface struct {
	ID              string              `json:"id"`
	Name            string              `json:"name"`
	Network         string              `json:"network"`
	Subnet          string              `json:"subnet"`
	InterfaceType   string              `json:"interface_type"`
	MACAddress      string              `json:"mac_address"`
	IPs             []InstanceNetworkIP `json:"ips"`
	ExternalDNSName string              `json:"external_dns_name"`
}

type InstanceNetworkIP struct {
	PrivateIPv4 *Address        `json:"private_ipv4,omitempty"`
	PublicIPv4  *PublicIPv4Info `json:"public_ipv4,omitempty"`
}

type PublicIPv4Info struct {
	Address string `json:"address"`
	ID      string `json:"id"`
	Type    string `json:"type"`
}

type Address struct {
	Address string `json:"address"`
}

type DiskAttachment struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	Size           string `json:"size"`
	Location       string `json:"location"`
	BlockSize      int    `json:"block_size"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	SerialNumber   string `json:"serial_number"`
	AttachmentType string `json:"attachment_type"` // "os" or "data"
	Mode           string `json:"mode"`            // "read-write" or "read-only"
}

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

type NetworkInterface struct {
	IPs []NetworkIP `json:"ips,omitempty"`
}

type NetworkIP struct {
	PublicIPv4 *PublicIPv4Config `json:"public_ipv4,omitempty"`
}

type PublicIPv4Config struct {
	Type string `json:"type"` // "static" or "dynamic"
}

type HostChannelAdapter struct {
	IBPartitionID string `json:"ib_partition_id"`
}

// Kept for compatibility
type DiskCreateRequest struct {
	SizeGiB int    `json:"size_gib"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
}

type CreateInstanceResponse struct {
	Operation InstanceOperation `json:"operation"`
}

type InstanceOperation struct {
	OperationID  string            `json:"operation_id"`
	State        string            `json:"state"`
	Metadata     OperationMetadata `json:"metadata"`
	Result       *json.RawMessage  `json:"result,omitempty"`
	Error        string            `json:"error,omitempty"`
	ErrorMessage string            `json:"error_message,omitempty"`
	StartedAt    string            `json:"started_at"`
	CompletedAt  string            `json:"completed_at,omitempty"`
}

// ErrorDetail returns a human-readable error message from the operation.
// It checks multiple fields where error details might be present.
func (op *InstanceOperation) ErrorDetail() string {
	if op.ErrorMessage != "" {
		return op.ErrorMessage
	}
	if op.Error != "" {
		return op.Error
	}
	if op.Result != nil {
		return string(*op.Result)
	}
	return ""
}

type OperationMetadata struct {
	OperationName string `json:"operation_name"`
	ID            string `json:"id"` // The VM ID
	Type          string `json:"type"`
}

type OperationStatus int

const (
	OperationStatusPending OperationStatus = iota
	OperationStatusSucceeded
	OperationStatusFailed
)

const PollInterval = 8 * time.Second

func (c *Client) GetVMOperationStatus(operationID string) (OperationStatus, *InstanceOperation, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/operations/%s", c.projectID, operationID)

	respBody, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return OperationStatusFailed, nil, err
	}

	log.Printf("[DEBUG] GetVMOperationStatus response: %s", string(respBody))

	var operation InstanceOperation
	if err := json.Unmarshal(respBody, &operation); err != nil {
		return OperationStatusFailed, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	state := strings.ToUpper(operation.State)
	log.Printf("[DEBUG] Operation %s state: %s", operationID, state)

	switch state {
	case "SUCCEEDED":
		return OperationStatusSucceeded, &operation, nil
	case "FAILED":
		log.Printf("[DEBUG] VM operation failed. Error: %s, ErrorMessage: %s, Result: %v",
			operation.Error, operation.ErrorMessage, operation.Result)
		return OperationStatusFailed, &operation, nil
	case "PENDING", "IN_PROGRESS":
		return OperationStatusPending, &operation, nil
	default:
		return OperationStatusFailed, &operation, fmt.Errorf("unknown operation state: %s", state)
	}
}

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
			time.Sleep(PollInterval)
		}
	}

	return false, nil, fmt.Errorf("operation timed out after %v", timeout)
}

func (c *Client) GetImageOperationStatus(operationID string) (OperationStatus, *InstanceOperation, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/custom-images/operations/%s", c.projectID, operationID)

	respBody, err := c.doRequest("GET", path, nil, nil)
	if err != nil {
		return OperationStatusFailed, nil, err
	}

	log.Printf("[DEBUG] GetImageOperationStatus response: %s", string(respBody))

	var operation InstanceOperation
	if err := json.Unmarshal(respBody, &operation); err != nil {
		return OperationStatusFailed, nil, fmt.Errorf("unmarshal response: %w", err)
	}

	state := strings.ToUpper(operation.State)
	log.Printf("[DEBUG] Image operation %s state: %s", operationID, state)

	switch state {
	case "SUCCEEDED":
		return OperationStatusSucceeded, &operation, nil
	case "FAILED":
		log.Printf("[DEBUG] Image operation failed. Error: %s, ErrorMessage: %s, Result: %v",
			operation.Error, operation.ErrorMessage, operation.Result)
		return OperationStatusFailed, &operation, nil
	case "PENDING", "IN_PROGRESS":
		return OperationStatusPending, &operation, nil
	default:
		return OperationStatusFailed, &operation, fmt.Errorf("unknown operation state: %s", state)
	}
}

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
			time.Sleep(PollInterval)
		}
	}

	return false, nil, fmt.Errorf("operation timed out after %v", timeout)
}

func (c *Client) CreateInstance(req *CreateInstanceRequest) (string, string, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances", c.projectID)

	respBody, err := c.doRequest("POST", path, req, nil)
	if err != nil {
		return "", "", err
	}

	log.Printf("[DEBUG] CreateInstance response: %s", string(respBody))

	var resp CreateInstanceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", "", fmt.Errorf("unmarshal response: %w", err)
	}

	if resp.Operation.Metadata.ID == "" {
		return "", "", fmt.Errorf("API did not return an instance ID. Response: %s", string(respBody))
	}
	if resp.Operation.OperationID == "" {
		return "", "", fmt.Errorf("API did not return an operation ID. Response: %s", string(respBody))
	}

	log.Printf("[DEBUG] Created instance ID: %s, operation ID: %s", resp.Operation.Metadata.ID, resp.Operation.OperationID)

	return resp.Operation.Metadata.ID, resp.Operation.OperationID, nil
}

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

type UpdateInstanceRequest struct {
	Action string `json:"action"`
}

func (c *Client) UpdateInstance(instanceID string, req *UpdateInstanceRequest) error {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/%s", c.projectID, instanceID)

	log.Printf("[DEBUG] UpdateInstance: %s with action: %s", instanceID, req.Action)
	_, err := c.doRequest("PATCH", path, req, nil)
	return err
}

func (c *Client) DeleteInstance(instanceID string) error {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/vms/instances/%s", c.projectID, instanceID)

	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}

type CustomImage struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	State       string `json:"state"`
	Location    string `json:"location"`
}

type CreateCustomImageRequest struct {
	DiskID      string `json:"DiskID"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Tags        string `json:"tags,omitempty"`
}

type CreateCustomImageResponse struct {
	Operation InstanceOperation `json:"operation"`
}

func (c *Client) CreateCustomImage(req *CreateCustomImageRequest) (string, error) {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/custom-images", c.projectID)

	respBody, err := c.doRequest("POST", path, req, nil)
	if err != nil {
		return "", err
	}

	var resp CreateCustomImageResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}

	return resp.Operation.OperationID, nil
}

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

func (c *Client) DeleteCustomImage(imageID string) error {
	path := fmt.Sprintf("/v1alpha5/projects/%s/compute/custom-images/%s", c.projectID, imageID)

	_, err := c.doRequest("DELETE", path, nil, nil)
	return err
}

type SSHKey struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

type CreateSSHKeyRequest struct {
	Name      string `json:"name"`
	PublicKey string `json:"public_key"`
}

type CreateSSHKeyResponse struct {
	SSHKey SSHKey `json:"ssh_key"`
}

func (c *Client) CreateSSHKey(req *CreateSSHKeyRequest) (*SSHKey, error) {
	respBody, err := c.doRequest("POST", "/v1alpha5/users/ssh-keys", req, nil)
	if err != nil {
		return nil, err
	}

	var resp CreateSSHKeyResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp.SSHKey, nil
}

func (c *Client) DeleteSSHKey(keyID string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/v1alpha5/users/ssh-keys/%s", keyID), nil, nil)
	return err
}
