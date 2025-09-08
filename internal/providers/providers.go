package providers

import "context"

type Node struct {
	Name    string
	IP      string
	ID      string
	SSHUser string
	SSHPort int
}

type Fleet struct {
	Name  string
	Nodes []Node
}

type CreateFleetRequest struct {
	Name      string
	Count     int
	Region    string
	Image     string
	Size      string
	Tags      []string
	SSHUser   string
	SSHKey    string
	CloudInit string
}

type Provider interface {
	Name() string
	CreateFleet(ctx context.Context, req CreateFleetRequest) (*Fleet, error)
	ListNodes(ctx context.Context, name string) ([]Node, error)
	DeleteFleet(ctx context.Context, name string) error
}
