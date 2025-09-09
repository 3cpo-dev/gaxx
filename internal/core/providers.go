package core

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LinodeProvider implements the Provider interface for Linode
type LinodeProvider struct {
	token  string
	client *http.Client
}

// NewLinodeProvider creates a new Linode provider
func NewLinodeProvider(token string) *LinodeProvider {
	return &LinodeProvider{
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// LinodeInstance represents a Linode instance
type LinodeInstance struct {
	ID     int      `json:"id"`
	Label  string   `json:"label"`
	IPv4   []string `json:"ipv4"`
	Status string   `json:"status"`
}

// LinodeCreateRequest represents the request to create a Linode instance
type LinodeCreateRequest struct {
	Region         string   `json:"region"`
	Type           string   `json:"type"`
	Image          string   `json:"image"`
	Label          string   `json:"label"`
	RootPass       string   `json:"root_pass"`
	Tags           []string `json:"tags"`
	AuthorizedKeys []string `json:"authorized_keys"`
	Booted         bool     `json:"booted"`
}

// CreateInstances creates multiple Linode instances
func (p *LinodeProvider) CreateInstances(ctx context.Context, count int, name string) ([]Instance, error) {
	instances := make([]Instance, 0, count)

	for i := 0; i < count; i++ {
		label := fmt.Sprintf("%s-%d", name, i+1)
		instance, err := p.createInstance(ctx, label)
		if err != nil {
			// Clean up already created instances
			p.cleanupInstances(ctx, instances)
			return nil, fmt.Errorf("create instance %d: %w", i+1, err)
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// createInstance creates a single Linode instance
func (p *LinodeProvider) createInstance(ctx context.Context, label string) (Instance, error) {
	req := LinodeCreateRequest{
		Region:         "us-east",
		Type:           "g6-nanode-1",
		Image:          "linode/ubuntu22.04",
		Label:          label,
		RootPass:       generatePassword(),
		Tags:           []string{"gaxx"},
		AuthorizedKeys: []string{}, // TODO: Add SSH key
		Booted:         true,
	}

	var linodeInst LinodeInstance
	if err := p.doRequest(ctx, "POST", "/linode/instances", req, &linodeInst); err != nil {
		return Instance{}, err
	}

	// Wait for instance to be running and get IP
	instance, err := p.waitForInstance(ctx, linodeInst.ID)
	if err != nil {
		return Instance{}, err
	}

	return instance, nil
}

// waitForInstance waits for a Linode instance to be ready
func (p *LinodeProvider) waitForInstance(ctx context.Context, instanceID int) (Instance, error) {
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return Instance{}, fmt.Errorf("timeout waiting for instance %d", instanceID)
		case <-ticker.C:
			var linodeInst LinodeInstance
			url := fmt.Sprintf("/linode/instances/%d", instanceID)
			if err := p.doRequest(ctx, "GET", url, nil, &linodeInst); err != nil {
				continue
			}

			if linodeInst.Status == "running" && len(linodeInst.IPv4) > 0 {
				return Instance{
					ID:   fmt.Sprintf("%d", linodeInst.ID),
					Name: linodeInst.Label,
					IP:   linodeInst.IPv4[0],
					User: "gx",
					Port: 22,
				}, nil
			}
		case <-ctx.Done():
			return Instance{}, ctx.Err()
		}
	}
}

// DeleteInstances deletes instances by name prefix
func (p *LinodeProvider) DeleteInstances(ctx context.Context, name string) error {
	instances, err := p.ListInstances(ctx, name)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		instanceID := instance.ID
		url := fmt.Sprintf("/linode/instances/%s", instanceID)
		if err := p.doRequest(ctx, "DELETE", url, nil, nil); err != nil {
			// Log error but continue with other instances
			fmt.Printf("Warning: failed to delete instance %s: %v\n", instanceID, err)
		}
	}

	return nil
}

// ListInstances lists instances by name prefix
func (p *LinodeProvider) ListInstances(ctx context.Context, name string) ([]Instance, error) {
	var response struct {
		Data []LinodeInstance `json:"data"`
	}

	if err := p.doRequest(ctx, "GET", "/linode/instances", nil, &response); err != nil {
		return nil, err
	}

	var instances []Instance
	for _, linodeInst := range response.Data {
		if name == "" || strings.HasPrefix(linodeInst.Label, name) {
			ip := ""
			if len(linodeInst.IPv4) > 0 {
				ip = linodeInst.IPv4[0]
			}
			instances = append(instances, Instance{
				ID:   fmt.Sprintf("%d", linodeInst.ID),
				Name: linodeInst.Label,
				IP:   ip,
				User: "gx",
				Port: 22,
			})
		}
	}

	return instances, nil
}

// doRequest performs an HTTP request to the Linode API with retry logic
func (p *LinodeProvider) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := "https://api.linode.com/v4" + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = strings.NewReader(string(jsonData))
	}

	// Retry logic for transient errors
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+p.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("do request: %w", err)
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Retry on rate limit or server errors
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				if attempt < maxRetries-1 {
					time.Sleep(time.Duration(attempt+1) * time.Second)
					continue
				}
			}
			return fmt.Errorf("linode api error %d: %s", resp.StatusCode, string(body))
		}

		if result != nil {
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				resp.Body.Close()
				return fmt.Errorf("decode response: %w", err)
			}
		}
		resp.Body.Close()
		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

// cleanupInstances deletes instances in case of partial failure
func (p *LinodeProvider) cleanupInstances(ctx context.Context, instances []Instance) {
	for _, instance := range instances {
		url := fmt.Sprintf("/linode/instances/%s", instance.ID)
		_ = p.doRequest(ctx, "DELETE", url, nil, nil)
	}
}

// generatePassword generates a random password
func generatePassword() string {
	// Simple password generation - in production, use crypto/rand
	return "GaxxTempPass123!"
}

// VultrProvider implements the Provider interface for Vultr
type VultrProvider struct {
	token  string
	client *http.Client
}

// NewVultrProvider creates a new Vultr provider
func NewVultrProvider(token string) *VultrProvider {
	return &VultrProvider{
		token: token,
		client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     90 * time.Second,
			},
		},
	}
}

// VultrInstance represents a Vultr instance
type VultrInstance struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	MainIP string `json:"main_ip"`
	Status string `json:"server_status"`
}

// CreateInstances creates multiple Vultr instances
func (p *VultrProvider) CreateInstances(ctx context.Context, count int, name string) ([]Instance, error) {
	instances := make([]Instance, 0, count)

	for i := 0; i < count; i++ {
		label := fmt.Sprintf("%s-%d", name, i+1)
		instance, err := p.createInstance(ctx, label)
		if err != nil {
			// Clean up already created instances
			p.cleanupInstances(ctx, instances)
			return nil, fmt.Errorf("create instance %d: %w", i+1, err)
		}
		instances = append(instances, instance)
	}

	return instances, nil
}

// createInstance creates a single Vultr instance
func (p *VultrProvider) createInstance(ctx context.Context, label string) (Instance, error) {
	req := map[string]interface{}{
		"region":      "ewr",
		"plan":        "vc2-1c-1gb",
		"os_id":       477, // Ubuntu 22.04
		"label":       label,
		"tag":         "gaxx",
		"enable_ipv6": false,
	}

	var vultrInst VultrInstance
	if err := p.doRequest(ctx, "POST", "/instances", req, &vultrInst); err != nil {
		return Instance{}, err
	}

	// Wait for instance to be running
	instance, err := p.waitForInstance(ctx, vultrInst.ID)
	if err != nil {
		return Instance{}, err
	}

	return instance, nil
}

// waitForInstance waits for a Vultr instance to be ready
func (p *VultrProvider) waitForInstance(ctx context.Context, instanceID string) (Instance, error) {
	timeout := time.After(10 * time.Minute)
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeout:
			return Instance{}, fmt.Errorf("timeout waiting for instance %s", instanceID)
		case <-ticker.C:
			var vultrInst VultrInstance
			url := fmt.Sprintf("/instances/%s", instanceID)
			if err := p.doRequest(ctx, "GET", url, nil, &vultrInst); err != nil {
				continue
			}

			if vultrInst.Status == "ok" && vultrInst.MainIP != "" {
				return Instance{
					ID:   vultrInst.ID,
					Name: vultrInst.Label,
					IP:   vultrInst.MainIP,
					User: "gx",
					Port: 22,
				}, nil
			}
		case <-ctx.Done():
			return Instance{}, ctx.Err()
		}
	}
}

// DeleteInstances deletes instances by name prefix
func (p *VultrProvider) DeleteInstances(ctx context.Context, name string) error {
	instances, err := p.ListInstances(ctx, name)
	if err != nil {
		return err
	}

	for _, instance := range instances {
		url := fmt.Sprintf("/instances/%s", instance.ID)
		if err := p.doRequest(ctx, "DELETE", url, nil, nil); err != nil {
			// Log error but continue with other instances
			fmt.Printf("Warning: failed to delete instance %s: %v\n", instance.ID, err)
		}
	}

	return nil
}

// ListInstances lists instances by name prefix
func (p *VultrProvider) ListInstances(ctx context.Context, name string) ([]Instance, error) {
	var response map[string]VultrInstance

	if err := p.doRequest(ctx, "GET", "/instances", nil, &response); err != nil {
		return nil, err
	}

	var instances []Instance
	for _, vultrInst := range response {
		if name == "" || strings.HasPrefix(vultrInst.Label, name) {
			instances = append(instances, Instance{
				ID:   vultrInst.ID,
				Name: vultrInst.Label,
				IP:   vultrInst.MainIP,
				User: "gx",
				Port: 22,
			})
		}
	}

	return instances, nil
}

// doRequest performs an HTTP request to the Vultr API with retry logic
func (p *VultrProvider) doRequest(ctx context.Context, method, path string, body interface{}, result interface{}) error {
	url := "https://api.vultr.com/v2" + path

	var reqBody io.Reader
	if body != nil {
		jsonData, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshal request: %w", err)
		}
		reqBody = strings.NewReader(string(jsonData))
	}

	// Retry logic for transient errors
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+p.token)
		req.Header.Set("Content-Type", "application/json")

		resp, err := p.client.Do(req)
		if err != nil {
			if attempt < maxRetries-1 {
				time.Sleep(time.Duration(attempt+1) * time.Second)
				continue
			}
			return fmt.Errorf("do request: %w", err)
		}

		if resp.StatusCode >= 400 {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()

			// Retry on rate limit or server errors
			if resp.StatusCode == 429 || resp.StatusCode >= 500 {
				if attempt < maxRetries-1 {
					time.Sleep(time.Duration(attempt+1) * time.Second)
					continue
				}
			}
			return fmt.Errorf("vultr api error %d: %s", resp.StatusCode, string(body))
		}

		if result != nil {
			if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
				resp.Body.Close()
				return fmt.Errorf("decode response: %w", err)
			}
		}
		resp.Body.Close()
		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

// cleanupInstances deletes instances in case of partial failure
func (p *VultrProvider) cleanupInstances(ctx context.Context, instances []Instance) {
	for _, instance := range instances {
		url := fmt.Sprintf("/instances/%s", instance.ID)
		_ = p.doRequest(ctx, "DELETE", url, nil, nil)
	}
}
