package linode

import (
	"context"

	prov "github.com/3cpo-dev/gaxx/internal/providers"
)

type Provider struct{ cfg prov.Config }

func New(cfg prov.Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) Name() string { return "linode" }

func (p *Provider) CreateFleet(ctx context.Context, req prov.CreateFleetRequest) (*prov.Fleet, error) {
	_ = ctx
	_ = req
	return &prov.Fleet{Name: req.Name, Nodes: []prov.Node{}}, nil
}

func (p *Provider) ListNodes(ctx context.Context, name string) ([]prov.Node, error) {
	_ = ctx
	_ = name
	return []prov.Node{}, nil
}

func (p *Provider) DeleteFleet(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	return nil
}
