package vultr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
	gssh "github.com/3cpo-dev/gaxx/internal/ssh"
)

type Provider struct{ cfg prov.Config }

func New(cfg prov.Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) Name() string { return "vultr" }

const vultrAPI = "https://api.vultr.com/v2"

type vultrInstance struct {
	ID     string `json:"id"`
	Label  string `json:"label"`
	MainIP string `json:"main_ip"`
	Status string `json:"status"`
}

type vultrListResp struct {
	Instances []vultrInstance `json:"instances"`
}

type vultrCreateReq struct {
	Region   string `json:"region"`
	Plan     string `json:"plan"`
	OSID     string `json:"os_id"`
	Label    string `json:"label"`
	UserData string `json:"user_data"`
}

type vultrCreateResp struct {
	Instance vultrInstance `json:"instance"`
}

func (p *Provider) token() (string, error) {
	t := p.cfg.Providers.Vultr.Token
	if t == "" {
		return "", fmt.Errorf("vultr token missing; set Providers.Vultr.Token or VULTR_TOKEN")
	}
	return t, nil
}

func (p *Provider) CreateFleet(ctx context.Context, req prov.CreateFleetRequest) (*prov.Fleet, error) {
	tok, err := p.token()
	if err != nil {
		return nil, err
	}
	region := firstNonEmpty(req.Region, p.cfg.Providers.Vultr.Region)
	plan := firstNonEmpty(req.Size, p.cfg.Providers.Vultr.Plan)
	osid := firstNonEmpty(req.Image, p.cfg.Providers.Vultr.OSID)
	user := firstNonEmpty(req.SSHUser, p.cfg.Defaults.User)
	sshKeyPath := p.cfg.SSH.KeyDir + "/id_ed25519"
	signer, err := gssh.LoadPrivateKeySigner(sshKeyPath)
	if err != nil {
		return nil, fmt.Errorf("load ssh key: %w", err)
	}
	pubAuth := string(gssh.MarshalAuthorized(signer))
	userData := prov.CloudInitUserData(user, pubAuth, "https://example.com/gaxx-agent")
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	fleet := &prov.Fleet{Name: req.Name}
	for i := 0; i < max(1, req.Count); i++ {
		label := fmt.Sprintf("%s-%d", req.Name, i+1)
		payload := vultrCreateReq{Region: region, Plan: plan, OSID: osid, Label: label, UserData: encodedUserData}
		var created vultrCreateResp
		if err := p.doJSON(ctx, tok, http.MethodPost, vultrAPI+"/instances", payload, &created); err != nil {
			return nil, fmt.Errorf("create instance: %w", err)
		}
		deadline := time.Now().Add(10 * time.Minute)
		for time.Now().Before(deadline) {
			var cur vultrInstance
			if err := p.doJSON(ctx, tok, http.MethodGet, vultrAPI+"/instances/"+created.Instance.ID, nil, &cur); err == nil {
				if cur.Status == "active" && cur.MainIP != "" {
					fleet.Nodes = append(fleet.Nodes, prov.Node{ID: cur.ID, Name: cur.Label, IP: cur.MainIP, SSHUser: user, SSHPort: p.cfg.Defaults.SSHPort})
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
	var list vultrListResp
	if err := p.doJSON(ctx, tok, http.MethodGet, vultrAPI+"/instances", nil, &list); err != nil {
		return nil, err
	}
	var nodes []prov.Node
	for _, inst := range list.Instances {
		if name != "" && !strings.HasPrefix(inst.Label, name) {
			continue
		}
		nodes = append(nodes, prov.Node{ID: inst.ID, Name: inst.Label, IP: inst.MainIP, SSHUser: p.cfg.Defaults.User, SSHPort: p.cfg.Defaults.SSHPort})
	}
	return nodes, nil
}

func (p *Provider) DeleteFleet(ctx context.Context, name string) error {
	tok, err := p.token()
	if err != nil {
		return err
	}
	var list vultrListResp
	if err := p.doJSON(ctx, tok, http.MethodGet, vultrAPI+"/instances", nil, &list); err != nil {
		return err
	}
	for _, inst := range list.Instances {
		if name == "" || strings.HasPrefix(inst.Label, name) {
			_ = p.doJSON(ctx, tok, http.MethodDelete, vultrAPI+"/instances/"+inst.ID, nil, nil)
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
		// Read the response body for more detailed error information
		var errorBody []byte
		errorBody, _ = io.ReadAll(resp.Body)
		return fmt.Errorf("vultr api status %d: %s", resp.StatusCode, string(errorBody))
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
