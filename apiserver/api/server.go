package api

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	erv1alpha1 "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned/typed/eventsrunner.io/v1alpha1"
	"k8s.io/client-go/rest"
)

// EventsRunnerAPI serves eventsrunner REST API via a configurable port.
// Supports jwt and/or mtls based authentication.
// For each request, the server will create Event CRDs.
type EventsRunnerAPI struct {
	*http.Server
	eventsRunnerClient *erv1alpha1.EventsrunnerV1alpha1Client
}

// AuthType is the type of authentication used by the server.
type AuthType string

const (
	// JWT is the authentication type for JWT based authentication.
	JWT AuthType = "jwt"
	// MTLS is the authentication type for MTLS based authentication.
	MTLS AuthType = "mtls"
	// None is the authentication type for no authentication.
	None AuthType = "none"
)

// MissingRequiredOptionError is returned when required options are missing.
type MissingRequiredOptionError struct {
	Option string
}

func (e MissingRequiredOptionError) Error() string {
	return fmt.Sprintf("missing required option: %s", e.Option)
}

// ServerOpts are options for creating a new eventsrunner-api server.
type ServerOpts struct {
	// Addr is the address to listen on.
	Addr string
	// AuthType is the type of authentication used by the server.
	AuthType AuthType
	// JWTSecret is the secret used to sign JWT tokens.
	JWTSecret string
	// Enables TLS.
	EnableTLS bool
	// CACertPath is the path to the CA certificate used for client authentication.
	CACert string
	// CertPath is the path to the server certificate.
	CertPath string
	// KeyPath is the path to the server key.
	KeyPath string
}

func loadTLSConfig(caCertPath, certPath, keyPath string) (*tls.Config, error) {
	caCert, err := ioutil.ReadFile(caCertPath)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, err
	}
	return &tls.Config{
		RootCAs:      caCertPool,
		Certificates: []tls.Certificate{cert},
	}, nil
}

// NewEventsRunnerAPI creates a new eventsrunner-api server.
func NewEventsRunnerAPI(kubeConfig *rest.Config, serverOpts ServerOpts) (*EventsRunnerAPI, error) {
	// check required options
	if serverOpts.Addr == "" {
		return nil, MissingRequiredOptionError{"Addr"}
	}
	if serverOpts.AuthType == "" {
		return nil, MissingRequiredOptionError{"AuthType"}
	}

	// server endpoint handler
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    serverOpts.Addr,
		Handler: mux,
	}
	// check jwt options
	if serverOpts.AuthType == JWT {
		if serverOpts.JWTSecret == "" {
			return nil, MissingRequiredOptionError{Option: "JWTSecret"}
		}
	}
	// check tls options
	if serverOpts.EnableTLS {
		if serverOpts.CACert == "" {
			return nil, MissingRequiredOptionError{"CACert"}
		}
		if serverOpts.CertPath == "" {
			return nil, MissingRequiredOptionError{"CertPath"}
		}
		if serverOpts.KeyPath == "" {
			return nil, MissingRequiredOptionError{"KeyPath"}
		}
		tlsConfig, err := loadTLSConfig(serverOpts.CACert, serverOpts.CertPath, serverOpts.KeyPath)
		if err != nil {
			return nil, err
		}
		if serverOpts.AuthType == MTLS {
			tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
		}
		server.TLSConfig = tlsConfig
	}

	// TODO: Register Handlers based on AuthType

	eventsRunnerClient, err := erv1alpha1.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	return &EventsRunnerAPI{
		eventsRunnerClient: eventsRunnerClient,
		Server:             server,
	}, nil
}
