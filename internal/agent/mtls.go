package agent

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/rs/zerolog/log"
)

// MTLSConfig holds mutual TLS configuration
type MTLSConfig struct {
	ServerCert   string
	ServerKey    string
	ClientCACert string
	RequireAuth  bool
}

// LoadMTLSConfig loads mTLS configuration from environment variables
func LoadMTLSConfig() MTLSConfig {
	return MTLSConfig{
		ServerCert:   os.Getenv("GAXX_AGENT_TLS_CERT"),
		ServerKey:    os.Getenv("GAXX_AGENT_TLS_KEY"),
		ClientCACert: os.Getenv("GAXX_AGENT_CLIENT_CA"),
		RequireAuth:  os.Getenv("GAXX_AGENT_REQUIRE_MTLS") == "true",
	}
}

// ConfigureTLS configures TLS for the HTTP server with optional mTLS
func (s *Server) ConfigureTLS(config MTLSConfig) (*tls.Config, error) {
	if config.ServerCert == "" || config.ServerKey == "" {
		return nil, fmt.Errorf("server cert and key required for TLS")
	}

	// Load server certificate
	cert, err := tls.LoadX509KeyPair(config.ServerCert, config.ServerKey)
	if err != nil {
		return nil, fmt.Errorf("load server certificate: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	// Configure client certificate validation if mTLS is enabled
	if config.RequireAuth && config.ClientCACert != "" {
		caCert, err := os.ReadFile(config.ClientCACert)
		if err != nil {
			return nil, fmt.Errorf("read client CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse client CA certificate")
		}

		tlsConfig.ClientCAs = caCertPool
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert

		log.Info().
			Str("ca_cert", config.ClientCACert).
			Msg("mTLS client authentication enabled")
	}

	return tlsConfig, nil
}

// MTLSMiddleware adds mTLS client certificate validation
func MTLSMiddleware(requireAuth bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if requireAuth && r.TLS != nil && len(r.TLS.PeerCertificates) == 0 {
				http.Error(w, "client certificate required", http.StatusUnauthorized)
				return
			}

			if len(r.TLS.PeerCertificates) > 0 {
				// Extract client information from certificate
				clientCert := r.TLS.PeerCertificates[0]
				r.Header.Set("X-Client-Subject", clientCert.Subject.String())
				r.Header.Set("X-Client-Serial", clientCert.SerialNumber.String())

				log.Debug().
					Str("subject", clientCert.Subject.String()).
					Str("serial", clientCert.SerialNumber.String()).
					Msg("mTLS client authenticated")
			}

			next.ServeHTTP(w, r)
		})
	}
}

// UpdateListenAndServeTLS updates the server to support TLS and mTLS
func (s *Server) ListenAndServeTLS(addr string, config MTLSConfig) error {
	tlsConfig, err := s.ConfigureTLS(config)
	if err != nil {
		return err
	}

	mux := http.NewServeMux()
	s.routes(mux)

	// Wrap with mTLS middleware
	handler := MTLSMiddleware(config.RequireAuth)(mux)

	s.srv = &http.Server{
		Addr:      addr,
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	log.Info().
		Str("addr", addr).
		Bool("mtls_required", config.RequireAuth).
		Msg("Starting agent with TLS/mTLS")

	return s.srv.ListenAndServeTLS("", "")
}
