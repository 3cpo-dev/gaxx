package localssh

import (
	"context"
	"fmt"

	"github.com/3cpo-dev/gaxx/internal/providers"
)

type Provider struct {
	cfg providers.Config
}

func New(cfg providers.Config) *Provider { return &Provider{cfg: cfg} }

func (p *Provider) Name() string { return "localssh" }

func (p *Provider) CreateFleet(ctx context.Context, req providers.CreateFleetRequest) (*providers.Fleet, error) {
	// No-op: we attach to existing hosts defined in config.
	nodes, err := p.ListNodes(ctx, req.Name)
	if err != nil {
		return nil, err
	}
	return &providers.Fleet{Name: req.Name, Nodes: nodes}, nil
}

func (p *Provider) ListNodes(ctx context.Context, name string) ([]providers.Node, error) {
	_ = ctx
	var nodes []providers.Node
	for _, h := range p.cfg.Providers.LocalSSH.Hosts {
		nodes = append(nodes, providers.Node{
			Name:    h.Name,
			IP:      h.IP,
			ID:      fmt.Sprintf("local-%s", h.Name),
			SSHUser: h.User,
			SSHPort: h.Port,
		})
	}
	return nodes, nil
}

func (p *Provider) DeleteFleet(ctx context.Context, name string) error {
	_ = ctx
	_ = name
	// No-op for local attachments.
	return nil
}
