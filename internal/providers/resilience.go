package providers

import (
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"time"

	"github.com/rs/zerolog/log"
)

// RetryConfig defines retry behavior for cloud provider operations
type RetryConfig struct {
	MaxRetries      int
	InitialDelay    time.Duration
	MaxDelay        time.Duration
	BackoffFactor   float64
	RetryableErrors []int // HTTP status codes that should be retried
}

// DefaultRetryConfig returns sensible retry defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries:      3,
		InitialDelay:    1 * time.Second,
		MaxDelay:        30 * time.Second,
		BackoffFactor:   2.0,
		RetryableErrors: []int{429, 500, 502, 503, 504}, // Rate limit + server errors
	}
}

// RateLimiter provides rate limiting for API calls
type RateLimiter struct {
	lastCall time.Time
	interval time.Duration
}

// NewRateLimiter creates a rate limiter with minimum interval between calls
func NewRateLimiter(requestsPerSecond float64) *RateLimiter {
	interval := time.Duration(float64(time.Second) / requestsPerSecond)
	return &RateLimiter{
		interval: interval,
	}
}

// Wait blocks until it's safe to make the next API call
func (rl *RateLimiter) Wait() {
	if rl.lastCall.IsZero() {
		rl.lastCall = time.Now()
		return
	}

	elapsed := time.Since(rl.lastCall)
	if elapsed < rl.interval {
		sleepTime := rl.interval - elapsed
		log.Debug().Dur("sleep", sleepTime).Msg("Rate limiting API call")
		time.Sleep(sleepTime)
	}
	rl.lastCall = time.Now()
}

// RetryableHTTPClient wraps HTTP client with retries and rate limiting
type RetryableHTTPClient struct {
	client      *http.Client
	retryConfig RetryConfig
	rateLimiter *RateLimiter
}

// NewRetryableHTTPClient creates a new HTTP client with retry logic
func NewRetryableHTTPClient(timeout time.Duration, requestsPerSecond float64) *RetryableHTTPClient {
	return &RetryableHTTPClient{
		client:      &http.Client{Timeout: timeout},
		retryConfig: DefaultRetryConfig(),
		rateLimiter: NewRateLimiter(requestsPerSecond),
	}
}

// Do executes HTTP request with retry logic and rate limiting
func (c *RetryableHTTPClient) Do(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= c.retryConfig.MaxRetries; attempt++ {
		// Rate limit before making request
		c.rateLimiter.Wait()

		// Clone request for retry (body might be consumed)
		reqClone := req.Clone(req.Context())

		resp, err := c.client.Do(reqClone)
		if err != nil {
			lastErr = err
			if attempt < c.retryConfig.MaxRetries {
				delay := c.calculateDelay(attempt)
				log.Warn().
					Err(err).
					Int("attempt", attempt+1).
					Int("max_retries", c.retryConfig.MaxRetries).
					Dur("delay", delay).
					Str("url", req.URL.String()).
					Msg("HTTP request failed, retrying")
				time.Sleep(delay)
				continue
			}
			return nil, lastErr
		}

		// Check if status code is retryable
		if c.shouldRetry(resp.StatusCode) && attempt < c.retryConfig.MaxRetries {
			resp.Body.Close()
			delay := c.calculateDelay(attempt)
			log.Warn().
				Int("status", resp.StatusCode).
				Int("attempt", attempt+1).
				Int("max_retries", c.retryConfig.MaxRetries).
				Dur("delay", delay).
				Str("url", req.URL.String()).
				Msg("HTTP request returned retryable error, retrying")
			time.Sleep(delay)
			continue
		}

		return resp, nil
	}

	return nil, lastErr
}

// shouldRetry determines if a status code should trigger a retry
func (c *RetryableHTTPClient) shouldRetry(statusCode int) bool {
	for _, code := range c.retryConfig.RetryableErrors {
		if statusCode == code {
			return true
		}
	}
	return false
}

// calculateDelay calculates exponential backoff delay with jitter
func (c *RetryableHTTPClient) calculateDelay(attempt int) time.Duration {
	delay := float64(c.retryConfig.InitialDelay) * math.Pow(c.retryConfig.BackoffFactor, float64(attempt))

	// Apply jitter (±25%)
	jitter := delay * 0.25 * (2*rand.Float64() - 1)
	delay += jitter

	// Cap at max delay
	if delay > float64(c.retryConfig.MaxDelay) {
		delay = float64(c.retryConfig.MaxDelay)
	}

	return time.Duration(delay)
}

// Paginator handles paginated API responses
type Paginator struct {
	PageSize   int
	MaxPages   int
	TotalCount int
}

// NewPaginator creates a paginator with sensible defaults
func NewPaginator() *Paginator {
	return &Paginator{
		PageSize: 100,
		MaxPages: 50, // Limit to prevent runaway pagination
	}
}

// ValidationError represents a validation error for cloud provider requests
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s=%s: %s", e.Field, e.Value, e.Message)
}

// CloudProviderValidator validates cloud provider requests
type CloudProviderValidator struct {
	validRegions map[string][]string // provider -> regions
	validImages  map[string][]string // provider -> images
	validSizes   map[string][]string // provider -> sizes
}

// NewCloudProviderValidator creates a validator with known valid values
func NewCloudProviderValidator() *CloudProviderValidator {
	return &CloudProviderValidator{
		validRegions: map[string][]string{
			"linode": {"us-east", "us-west", "eu-west", "ap-south", "ap-southeast", "eu-central"},
			"vultr":  {"ewr", "sea", "lax", "atl", "ams", "lon", "fra", "sgp", "nrt"},
		},
		validImages: map[string][]string{
			"linode": {"linode/ubuntu22.04", "linode/ubuntu20.04", "linode/debian11", "linode/centos7"},
			"vultr":  {"387", "477", "215", "230"}, // Ubuntu 20.04, 22.04, Debian 11, CentOS 7
		},
		validSizes: map[string][]string{
			"linode": {"g6-nanode-1", "g6-standard-1", "g6-standard-2", "g6-standard-4"},
			"vultr":  {"vc2-1c-1gb", "vc2-1c-2gb", "vc2-2c-2gb", "vc2-2c-4gb"},
		},
	}
}

// ValidateCreateRequest validates a fleet creation request
func (v *CloudProviderValidator) ValidateCreateRequest(provider string, req CreateFleetRequest) error {
	if req.Name == "" {
		return ValidationError{Field: "name", Value: "", Message: "fleet name is required"}
	}

	if req.Count <= 0 || req.Count > 100 {
		return ValidationError{Field: "count", Value: fmt.Sprintf("%d", req.Count), Message: "count must be between 1 and 100"}
	}

	if req.Region != "" {
		if err := v.validateRegion(provider, req.Region); err != nil {
			return err
		}
	}

	if req.Image != "" {
		if err := v.validateImage(provider, req.Image); err != nil {
			return err
		}
	}

	if req.Size != "" {
		if err := v.validateSize(provider, req.Size); err != nil {
			return err
		}
	}

	return nil
}

func (v *CloudProviderValidator) validateRegion(provider, region string) error {
	validRegions, exists := v.validRegions[provider]
	if !exists {
		return nil // Skip validation for unknown providers
	}

	for _, valid := range validRegions {
		if region == valid {
			return nil
		}
	}

	return ValidationError{
		Field:   "region",
		Value:   region,
		Message: fmt.Sprintf("invalid region for %s. Valid regions: %v", provider, validRegions),
	}
}

func (v *CloudProviderValidator) validateImage(provider, image string) error {
	validImages, exists := v.validImages[provider]
	if !exists {
		return nil // Skip validation for unknown providers
	}

	for _, valid := range validImages {
		if image == valid {
			return nil
		}
	}

	return ValidationError{
		Field:   "image",
		Value:   image,
		Message: fmt.Sprintf("invalid image for %s. Valid images: %v", provider, validImages),
	}
}

func (v *CloudProviderValidator) validateSize(provider, size string) error {
	validSizes, exists := v.validSizes[provider]
	if !exists {
		return nil // Skip validation for unknown providers
	}

	for _, valid := range validSizes {
		if size == valid {
			return nil
		}
	}

	return ValidationError{
		Field:   "size",
		Value:   size,
		Message: fmt.Sprintf("invalid size for %s. Valid sizes: %v", provider, validSizes),
	}
}
