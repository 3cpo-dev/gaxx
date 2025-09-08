package linode

import (
	"context"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
	// Future imports for full implementation:
	// "fmt"
	// "time"
	// linodego SDK:
	// linodego "github.com/linode/linodego"
	// "golang.org/x/oauth2"
)

type Provider struct{ cfg prov.Config }

func New(cfg prov.Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) Name() string { return "linode" }

func (p *Provider) CreateFleet(ctx context.Context, req prov.CreateFleetRequest) (*prov.Fleet, error) {
	// End-to-end outline (to be implemented):
	// 1) OAuth2 token from cfg, create linodego client
	// 2) Choose region/type/image from req or defaults
	// 3) Build cloud-init user data using providers.CloudInitUserData
	// 4) Create instances; collect IDs
	// 5) Poll until running and public IP available
	// 6) Map into []prov.Node and return Fleet
	_ = ctx
	_ = req
	return &prov.Fleet{Name: req.Name, Nodes: []prov.Node{}}, nil
}

func (p *Provider) ListNodes(ctx context.Context, name string) ([]prov.Node, error) {
	// Outline: list instances (optionally by tag/prefix), map to []prov.Node
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
