package api

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"

	middleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	erv1alpha1Client "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned/typed/eventsrunner.io/v1alpha1"
	"k8s.io/client-go/rest"
)

// EventsRunnerAPI serves eventsrunner REST API via a configurable port.
// Supports jwt and/or mtls based authentication.
// For each request, the server will create Event CRDs.
type EventsRunnerAPI struct {
	server             *http.Server
	healthzServer      *http.Server
	eventsRunnerClient *erv1alpha1Client.EventsrunnerV1alpha1Client
	namespace          string
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
	Addr        string `default:"0.0.0.0"`
	Port        int    `default:"8080"`
	HealthzPort int    `default:"8081"`
	// Namespace is the namespace to create Event CRDs in.
	Namespace string `default:"eventsrunner"`
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

// loadTLSConfig loads relevant certificates and returns a TLS config.
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
	if serverOpts.Port == 0 {
		return nil, MissingRequiredOptionError{"Port"}
	}
	if serverOpts.HealthzPort == 0 {
		return nil, MissingRequiredOptionError{"HealthzPort"}
	}
	if serverOpts.AuthType == "" {
		return nil, MissingRequiredOptionError{"AuthType"}
	}
	if serverOpts.Namespace == "" {
		return nil, MissingRequiredOptionError{"Namespace"}
	}

	healthzMux := http.NewServeMux()
	healthzMux.HandleFunc("/healthz", healthzHandler)
	healthzServer := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverOpts.Addr, serverOpts.HealthzPort),
		Handler: healthzMux,
	}

	// server endpoint handler
	mux := http.NewServeMux()
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", serverOpts.Addr, serverOpts.Port),
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

	eventsRunnerClient, err := erv1alpha1Client.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	eventsRunnerAPI := &EventsRunnerAPI{
		server:             server,
		healthzServer:      healthzServer,
		eventsRunnerClient: eventsRunnerClient,
	}

	if serverOpts.AuthType == JWT {
		keyFunc := func(ctx context.Context) (interface{}, error) {
			return []byte(serverOpts.JWTSecret), nil
		}
		jwtValidator, err := validator.New(
			keyFunc,
			validator.HS256,
			"https://eventsrunner.io",
			[]string{"sensors.eventsrunner.io"},
		)
		if err != nil {
			return nil, err
		}
		jwtmiddleware := middleware.New(jwtValidator.ValidateToken)
		mux.Handle("/api/v1/events", jwtmiddleware.CheckJWT(http.HandlerFunc(eventsRunnerAPI.eventPostHandler)))
	} else {
		mux.HandleFunc("/api/v1/events", eventsRunnerAPI.eventPostHandler)
	}
	return eventsRunnerAPI, nil
}

type eventRequestBody struct {
	EventType  string        `json:"eventType"`
	ResourceID string        `json:"resourceID"`
	RuleID     string        `json:"ruleID"`
	EventData  []interface{} `json:"eventData"`
}

func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func (erapi *EventsRunnerAPI) eventPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// Parse Body
	// Create Event 
	w.WriteHeader(http.StatusOK)
}
