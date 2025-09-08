package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	xssh "golang.org/x/crypto/ssh"
)

type Dialer interface {
	Dial(network, addr string) (net.Conn, error)
}

type NetDialer struct{ Timeout time.Duration }

func (d NetDialer) Dial(network, addr string) (net.Conn, error) {
	nd := &net.Dialer{Timeout: d.Timeout}
	return nd.Dial(network, addr)
}

type Client struct {
	Addr       string
	User       string
	Signer     xssh.Signer
	KnownHosts xssh.HostKeyCallback
	Timeout    time.Duration
	Retries    int
	Backoff    time.Duration
	Dialer     Dialer
}

func (c *Client) makeConfig() (*xssh.ClientConfig, error) {
	if c.Signer == nil {
		return nil, errors.New("ssh: signer required")
	}
	if c.KnownHosts == nil {
		c.KnownHosts = xssh.InsecureIgnoreHostKey() // replaced by strict callback by caller normally
	}
	return &xssh.ClientConfig{
		User:            c.User,
		Auth:            []xssh.AuthMethod{xssh.PublicKeys(c.Signer)},
		HostKeyCallback: c.KnownHosts,
		Timeout:         c.Timeout,
	}, nil
}

// RunCommand executes a remote command with retries and basic backoff.
func (c *Client) RunCommand(ctx context.Context, command string) (string, string, error) {
	cfg, err := c.makeConfig()
	if err != nil {
		return "", "", err
	}
	var lastErr error
	retries := c.Retries
	if retries < 0 {
		retries = 0
	}
	backoff := c.Backoff
	if backoff <= 0 {
		backoff = 500 * time.Millisecond
	}
	for attempt := 0; attempt <= retries; attempt++ {
		select {
		case <-ctx.Done():
			return "", "", ctx.Err()
		default:
		}
		cli, err := xssh.Dial("tcp", c.Addr, cfg)
		if err != nil {
			lastErr = err
		} else {
			session, err := cli.NewSession()
			if err == nil {
				defer session.Close()
				stdout, err := session.Output(command)
				if err == nil {
					return string(stdout), "", nil
				}
				// If Output fails, try CombinedOutput for broader error context
				combined, cErr := session.CombinedOutput(command)
				if cErr == nil {
					return string(combined), "", nil
				}
				lastErr = fmt.Errorf("run command: %w", err)
			} else {
				lastErr = fmt.Errorf("new session: %w", err)
			}
			_ = cli.Close()
		}
		if attempt < retries {
			select {
			case <-ctx.Done():
				return "", "", ctx.Err()
			case <-time.After(backoff * time.Duration(attempt+1)):
			}
		}
	}
	return "", "", lastErr
}

// Dial establishes an SSH connection using the provided client configuration.
// The caller is responsible for closing the returned client.
func Dial(ctx context.Context, c *Client) (*xssh.Client, error) {
    cfg, err := c.makeConfig()
    if err != nil {
        return nil, err
    }
    type res struct {
        cli *xssh.Client
        err error
    }
    ch := make(chan res, 1)
    go func() {
        cli, err := xssh.Dial("tcp", c.Addr, cfg)
        ch <- res{cli: cli, err: err}
    }()
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    case r := <-ch:
        return r.cli, r.err
    }
}
