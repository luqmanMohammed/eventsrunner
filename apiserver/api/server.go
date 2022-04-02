package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	middleware "github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/validator"
	erAPI "github.com/luqmanMohammed/eventsrunner/crd/pkg/apis/eventsrunner.io/v1alpha1"
	erClient "github.com/luqmanMohammed/eventsrunner/crd/pkg/client/clientset/versioned/typed/eventsrunner.io/v1alpha1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
)

// EventsRunnerAPI serves eventsrunner REST API via a configurable port.
// Supports jwt and/or mtls based authentication.
// For each request, the server will create Event CRDs.
type EventsRunnerAPI struct {
	server             *http.Server
	healthzServer      *http.Server
	eventsRunnerClient *erClient.EventsrunnerV1alpha1Client
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
// TODO: Group options together.
type ServerOpts struct {
	// Addr is the address to listen on.
	Addr        string
	Port        int
	HealthzPort int
	// Namespace is the namespace to create Event CRDs in.
	Namespace string
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
// TODO: Create generic function for validating options
// TODO: Add tests for both jwt and mtls based auth
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

	eventsRunnerClient, err := erClient.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}

	eventsRunnerAPI := &EventsRunnerAPI{
		server:             server,
		healthzServer:      healthzServer,
		eventsRunnerClient: eventsRunnerClient,
		namespace:          serverOpts.Namespace,
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

// Start initializes the required components and starts both api and
// api-healthz servers.
// In case of any server start errors, function will panic.
func (erapi *EventsRunnerAPI) Start() {
	klog.V(1).Infof("Starting eventsrunner-api server on %s", erapi.server.Addr)
	klog.V(1).Infof("Starting eventsrunner-api healthz server on %s", erapi.healthzServer.Addr)
	wg := sync.WaitGroup{}
	wg.Add(2)
	startServer := func(wg *sync.WaitGroup, server *http.Server) {
		defer wg.Done()
		if err := server.ListenAndServe(); err != nil {
			if err != http.ErrServerClosed {
				klog.V(1).Info("Server successfully stopped")
			}
			klog.Fatalf("EventsRunnerAPI: ListenAndServe() failed: %v", err)
		}
	}
	go startServer(&wg, erapi.server)
	go startServer(&wg, erapi.healthzServer)
	wg.Wait()
}

// healthzResponse is the response body for the /healthz endpoint.
var healthzResponse map[string]string = map[string]string{
	"status": "ok",
}

// healthzHandler handles /healthz requests.
// It returns a 200 OK response with the status "ok" in the body.
func healthzHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(healthzResponse); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
}

// TODO: Add methods to stop the server gracefully
// TODO: Add method to listen for signals and gracefully stop the server

// validationError is returned when the request is invalid.
// validationError can be caused by missing required fields or invalid values.
type validationError struct {
	Msg string
}

// Error returns a string representation of the error.
// Implements the error interface.
func (err validationError) Error() string {
	return fmt.Sprintf("validation error: %s", err.Msg)
}

// eventPostRequestBody models the request body for the /api/v1/events endpoint.
// Used to parse and validate the request body.
type eventPostRequestBody struct {
	EventType  erAPI.EventType          `json:"eventType"`
	ResourceID string                   `json:"resourceID"`
	RuleID     string                   `json:"ruleID"`
	EventData  []map[string]interface{} `json:"eventData"`
}

// validate validates the eventPostRequestBody to make
// sure required fields are present.
// Confirms following fields are present:
// - eventType
// - resourceID
// - ruleID
// Confirms following fields are valid:
// - eventType (added, updated, deleted)
func (eprb eventPostRequestBody) validate() error {
	if eprb.EventType == "" {
		return validationError{Msg: "eventType is required"}
	}
	if eprb.ResourceID == "" {
		return validationError{Msg: "resourceID is required"}
	}
	if eprb.RuleID == "" {
		return validationError{Msg: "ruleID is required"}
	}
	if eprb.EventType != erAPI.ADDED && eprb.EventType != erAPI.UPDATED && eprb.EventType != erAPI.DELETED {
		return validationError{Msg: "eventType must be one of ADDED, UPDATED, or DELETED"}
	}
	return nil
}

// eventPostResponseBody models the response body for the /api/v1/events endpoint.
type eventPostResponseBody struct {
	Error        string `json:"error,omitempty"`
	Message      string `json:"message,omitempty"`
	ResourceName string `json:"resourceName,omitempty"`
}

// handleEventPostRequest helps to handle response to the POST requests to the 
// /api/v1/events endpoint. 
func handleEventPostResponse(w http.ResponseWriter, responseBody eventPostResponseBody, responseCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(responseCode)
	if err := json.NewEncoder(w).Encode(responseBody); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// TODO: Add tests
// TODO: Benchmark performance
func (erapi *EventsRunnerAPI) eventPostHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		handleEventPostResponse(w, eventPostResponseBody{
			Error: "Method not allowed",
		}, http.StatusMethodNotAllowed)
		return
	}
	var eventRequestBody eventPostRequestBody
	if err := json.NewDecoder(r.Body).Decode(&eventRequestBody); err != nil {
		handleEventPostResponse(w, eventPostResponseBody{
			Error: err.Error(),
		}, http.StatusBadRequest)
	}
	if err := eventRequestBody.validate(); err != nil {
		handleEventPostResponse(w, eventPostResponseBody{
			Error: err.Error(),
		}, http.StatusBadRequest)
		return
	}
	eventData := make([]string, len(eventRequestBody.EventData))
	for i, event := range eventRequestBody.EventData {
		buffer := bytes.NewBuffer(nil)
		if err := json.NewEncoder(buffer).Encode(event); err != nil {
			handleEventPostResponse(w, eventPostResponseBody{
				Error: err.Error(),
			}, http.StatusBadRequest)
			return
		}
		eventData[i] = buffer.String()
	}
	eventObj := &erAPI.Event{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: eventRequestBody.ResourceID + "-",
			Namespace:    erapi.namespace,
			Labels: map[string]string{
				"er.io/rule-id": eventRequestBody.RuleID,
			},
		},
		Spec: erAPI.EventSpec{
			ResourceID: eventRequestBody.ResourceID,
			RuleID:     eventRequestBody.RuleID,
			EventType:  eventRequestBody.EventType,
			EventData:  eventData,
		},
	}
	event, err := erapi.eventsRunnerClient.Events(erapi.namespace).Create(context.TODO(), eventObj, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsTooManyRequests(err) {
			handleEventPostResponse(w, eventPostResponseBody{
				Error: "kube-api throttled the request",
			}, http.StatusTooManyRequests)
			return
		}
		handleEventPostResponse(w, eventPostResponseBody{
			Error: err.Error(),
		}, http.StatusInternalServerError)
		return
	}
	handleEventPostResponse(w, eventPostResponseBody{
		ResourceName: event.Name,
		Message:      "Event created successfully",
	}, http.StatusOK)
}
