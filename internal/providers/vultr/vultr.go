package vultr

import (
	"context"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
	// Future imports for full implementation:
	// "fmt"
	// "time"
	// vultr SDK:
	// "github.com/vultr/govultr/v3"
	// "golang.org/x/oauth2"
)

type Provider struct{ cfg prov.Config }

func New(cfg prov.Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) Name() string { return "vultr" }

func (p *Provider) CreateFleet(ctx context.Context, req prov.CreateFleetRequest) (*prov.Fleet, error) {
	// End-to-end outline (to be implemented):
	// 1) Token from cfg, create client
	// 2) Choose region/plan/os_id from req or defaults
	// 3) Build cloud-init user data (providers.CloudInitUserData)
	// 4) Create instances; collect IDs
	// 5) Poll until active and public IP available
	// 6) Map into []prov.Node
	_ = ctx
	_ = req
	return &prov.Fleet{Name: req.Name, Nodes: []prov.Node{}}, nil
}

func (p *Provider) ListNodes(ctx context.Context, name string) ([]prov.Node, error) {
	// Outline: list instances (optionally by tag/label), map to []prov.Node
	_ = ctx
	_ = name
	return []prov.Node{}, nil
}

func (p *Provider) DeleteFleet(ctx context.Context, name string) error {
	// Outline: find instances by name prefix and delete
	_ = ctx
	_ = name
	return nil
}
