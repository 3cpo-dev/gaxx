package linode

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
	gssh "github.com/3cpo-dev/gaxx/internal/ssh"
)

type Provider struct {
	cfg       prov.Config
	client    *prov.RetryableHTTPClient
	validator *prov.CloudProviderValidator
}

func New(cfg prov.Config) *Provider {
	return &Provider{
		cfg:       cfg,
		client:    prov.NewRetryableHTTPClient(30*time.Second, 2.0), // 2 req/sec for Linode
		validator: prov.NewCloudProviderValidator(),
	}
}

func (p *Provider) Name() string { return "linode" }

const linodeAPI = "https://api.linode.com/v4"

type linodeInstance struct {
	ID     int      `json:"id"`
	Label  string   `json:"label"`
	IPv4   []string `json:"ipv4"`
	Status string   `json:"status"`
}

type linodeCreateReq struct {
	Region         string          `json:"region"`
	Type           string          `json:"type"`
	Image          string          `json:"image"`
	Label          string          `json:"label"`
	RootPass       string          `json:"root_pass"`
	Tags           []string        `json:"tags,omitempty"`
	AuthorizedKeys []string        `json:"authorized_keys,omitempty"`
	Metadata       *linodeMetadata `json:"metadata,omitempty"`
	Booted         bool            `json:"booted"`
}

type linodeMetadata struct {
	UserData string `json:"user_data"`
}

type linodeCreateResp linodeInstance

type linodeListResp struct {
	Data []linodeInstance `json:"data"`
}

func (p *Provider) token() (string, error) {
	t := p.cfg.Providers.Linode.Token
	if t == "" {
		return "", fmt.Errorf("linode token missing; set Providers.Linode.Token or LINODE_TOKEN")
	}
	return t, nil
}

func randPass() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

func (p *Provider) CreateFleet(ctx context.Context, req prov.CreateFleetRequest) (*prov.Fleet, error) {
	tok, err := p.token()
	if err != nil {
		return nil, err
	}
	region := firstNonEmpty(req.Region, p.cfg.Providers.Linode.Region)
	typeID := firstNonEmpty(req.Size, p.cfg.Providers.Linode.Type)
	image := firstNonEmpty(req.Image, p.cfg.Providers.Linode.Image)
	user := firstNonEmpty(req.SSHUser, p.cfg.Defaults.User)
	sshKeyPath := p.cfg.SSH.KeyDir + "/id_ed25519"
	signer, err := gssh.LoadPrivateKeySigner(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load ssh key: %w", err)
	}
	pubAuth := string(gssh.MarshalAuthorized(signer))
	userData := prov.CloudInitUserData(user, pubAuth, "https://example.com/gaxx-agent")
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))
	tags := append([]string{"gaxx"}, p.cfg.Providers.Linode.Tags...)

	fleet := &prov.Fleet{Name: req.Name}
	for i := 0; i < max(1, req.Count); i++ {
		label := fmt.Sprintf("%s-%d", req.Name, i+1)
		payload := linodeCreateReq{
			Region:         region,
			Type:           typeID,
			Image:          image,
			Label:          label,
			RootPass:       randPass(),
			Tags:           tags,
			AuthorizedKeys: []string{pubAuth},
			Metadata:       &linodeMetadata{UserData: encodedUserData},
			Booted:         true,
		}
		var created linodeCreateResp
		if err := p.doJSON(ctx, tok, http.MethodPost, linodeAPI+"/linode/instances", payload, &created); err != nil {
			return nil, fmt.Errorf("create instance: %w", err)
		}
		// Poll until running with IP
		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			var cur linodeInstance
			if err := p.doJSON(ctx, tok, http.MethodGet, fmt.Sprintf(linodeAPI+"/linode/instances/%d", created.ID), nil, &cur); err == nil {
				if cur.Status == "running" && len(cur.IPv4) > 0 {
					fleet.Nodes = append(fleet.Nodes, prov.Node{ID: fmt.Sprintf("%d", cur.ID), Name: cur.Label, IP: cur.IPv4[0], SSHUser: user, SSHPort: p.cfg.Defaults.SSHPort})
					break
				}
			}
			time.Sleep(5 * time.Second)
		}
	}
	return fleet, nil
}

func (p *Provider) ListNodes(ctx context.Context, name string) ([]prov.Node, error) {
	tok, err := p.token()
	if err != nil {
		return nil, err
	}
	var list linodeListResp
	if err := p.doJSON(ctx, tok, http.MethodGet, linodeAPI+"/linode/instances", nil, &list); err != nil {
		return nil, err
	}
	var nodes []prov.Node
	for _, inst := range list.Data {
		if name != "" && !strings.HasPrefix(inst.Label, name) {
			continue
		}
		ip := ""
		if len(inst.IPv4) > 0 {
			ip = inst.IPv4[0]
		}
		nodes = append(nodes, prov.Node{ID: fmt.Sprintf("%d", inst.ID), Name: inst.Label, IP: ip, SSHUser: p.cfg.Defaults.User, SSHPort: p.cfg.Defaults.SSHPort})
	}
	return nodes, nil
}

func (p *Provider) DeleteFleet(ctx context.Context, name string) error {
	tok, err := p.token()
	if err != nil {
		return err
	}
	var list linodeListResp
	if err := p.doJSON(ctx, tok, http.MethodGet, linodeAPI+"/linode/instances", nil, &list); err != nil {
		return err
	}
	for _, inst := range list.Data {
		if name == "" || strings.HasPrefix(inst.Label, name) {
			_ = p.doJSON(ctx, tok, http.MethodDelete, fmt.Sprintf(linodeAPI+"/linode/instances/%d", inst.ID), nil, nil)
		}
	}
	return nil
}

func (p *Provider) doJSON(ctx context.Context, token, method, url string, body interface{}, out interface{}) error {
	var req *http.Request
	var err error
	if body != nil {
		buf, e := json.Marshal(body)
		if e != nil {
			return e
		}
		req, err = http.NewRequestWithContext(ctx, method, url, strings.NewReader(string(buf)))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 && method != http.MethodDelete {
		return fmt.Errorf("linode api status %d", resp.StatusCode)
	}
	if out != nil {
		dec := json.NewDecoder(resp.Body)
		return dec.Decode(out)
	}
	return nil
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
