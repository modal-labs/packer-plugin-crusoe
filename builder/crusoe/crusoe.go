package crusoe

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client represents a Crusoe API client
type Client struct {
	accessKeyID     string
	secretAccessKey string
	apiEndpoint     string
	httpClient      *http.Client
}

// NewClient creates a new Crusoe API client
func NewClient(accessKeyID, secretAccessKey, apiEndpoint string) *Client {
	return &Client{
		accessKeyID:     accessKeyID,
		secretAccessKey: secretAccessKey,
		apiEndpoint:     apiEndpoint,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// doRequest performs an HTTP request with authentication
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonData)
	}

	url := fmt.Sprintf("%s%s", c.apiEndpoint, path)
	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.secretAccessKey))

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
		return nil, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
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
	Name      string              `json:"name"`
	Type      string              `json:"type"`
	Location  string              `json:"location"`
	Image     string              `json:"image"`
	SSHKeys   []string            `json:"ssh_keys,omitempty"`
	UserData  string              `json:"user_data,omitempty"`
	Tags      []string            `json:"tags,omitempty"`
	NetworkID string              `json:"network_id,omitempty"`
	SubnetID  string              `json:"subnet_id,omitempty"`
	Disks     []DiskCreateRequest `json:"disks,omitempty"`
}

// DiskCreateRequest represents a disk creation request
type DiskCreateRequest struct {
	SizeGiB int    `json:"size_gib"`
	Type    string `json:"type"`
	Mode    string `json:"mode"`
}

// CreateInstanceResponse represents the response from creating an instance
type CreateInstanceResponse struct {
	Instance Instance `json:"instance"`
}

// CreateInstance creates a new VM instance
func (c *Client) CreateInstance(req *CreateInstanceRequest) (*Instance, error) {
	respBody, err := c.doRequest("POST", "/v1alpha5/compute/instances", req)
	if err != nil {
		return nil, err
	}

	var resp CreateInstanceResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	return &resp.Instance, nil
}

// GetInstance retrieves an instance by ID
func (c *Client) GetInstance(instanceID string) (*Instance, error) {
	respBody, err := c.doRequest("GET", fmt.Sprintf("/v1alpha5/compute/instances/%s", instanceID), nil)
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
	_, err := c.doRequest("PATCH", fmt.Sprintf("/v1alpha5/compute/instances/%s", instanceID), req)
	return err
}

// DeleteInstance deletes an instance
func (c *Client) DeleteInstance(instanceID string) error {
	_, err := c.doRequest("DELETE", fmt.Sprintf("/v1alpha5/compute/instances/%s", instanceID), nil)
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
	respBody, err := c.doRequest("POST", "/v1alpha5/compute/custom-images", req)
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
	respBody, err := c.doRequest("GET", fmt.Sprintf("/v1alpha5/compute/custom-images/%s", imageID), nil)
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
	_, err := c.doRequest("DELETE", fmt.Sprintf("/v1alpha5/compute/custom-images/%s", imageID), nil)
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
	respBody, err := c.doRequest("POST", "/v1alpha5/ssh-keys", req)
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
	_, err := c.doRequest("DELETE", fmt.Sprintf("/v1alpha5/ssh-keys/%s", keyID), nil)
	return err
}
