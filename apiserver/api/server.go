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
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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

// MissingRequiredOptionError is returned when required configuration options are missing.
type MissingRequiredOptionError struct {
	Option string
}

// Error returns a string representation of the error.
// Implements the error interface.
func (e MissingRequiredOptionError) Error() string {
	return fmt.Sprintf("missing required option: %s", e.Option)
}

// InvalidOptionError is returned when the configuration options are invalid.
type InvalidOptionError struct {
	OptionName   string
	OptionValue  interface{}
	ValidOptions []interface{}
}

func (iop InvalidOptionError) Error() string {
	return fmt.Sprintf("invalid option: %s=%v, valid options: %v", iop.OptionName, iop.OptionValue, iop.ValidOptions)
}

// EventsRunnerAPIServerOpts are options for creating a new eventsrunner-api server.
// TODO: Group options together.
type EventsRunnerAPIServerOpts struct {
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

// validate validates the configuration options.
// Confirms all required options are present.
// Confirms authType is valid.
func (erOpts *EventsRunnerAPIServerOpts) validate() error {
	if erOpts.Addr == "" {
		return MissingRequiredOptionError{"Addr"}
	}
	if erOpts.Port == 0 {
		return MissingRequiredOptionError{"Port"}
	}
	if erOpts.HealthzPort == 0 {
		return MissingRequiredOptionError{"HealthzPort"}
	}
	if erOpts.AuthType == "" {
		return MissingRequiredOptionError{"AuthType"}
	}
	if erOpts.Namespace == "" {
		return MissingRequiredOptionError{"Namespace"}
	}
	if erOpts.AuthType == JWT && erOpts.JWTSecret == "" {
		return MissingRequiredOptionError{"JWTSecret"}
	}
	if erOpts.EnableTLS {
		if erOpts.CACert == "" {
			return MissingRequiredOptionError{"CACert"}
		}
		if erOpts.CertPath == "" {
			return MissingRequiredOptionError{"CertPath"}
		}
		if erOpts.KeyPath == "" {
			return MissingRequiredOptionError{"KeyPath"}
		}
	}
	if erOpts.AuthType != JWT && erOpts.AuthType != MTLS && erOpts.AuthType != None {
		return InvalidOptionError{
			OptionName:  "AuthType",
			OptionValue: erOpts.AuthType,
			ValidOptions: []interface{}{
				JWT,
				MTLS,
				None,
			},
		}
	}
	return nil
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
// TODO: Add tests for both jwt and mtls based auth
func NewEventsRunnerAPI(kubeConfig *rest.Config, serverOpts EventsRunnerAPIServerOpts) (*EventsRunnerAPI, error) {
	if err := serverOpts.validate(); err != nil {
		return nil, err
	}

	// creates server to handler healthz endpoint
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

	// if TLS is enabled then load the certificates and create a TLS config.
	if serverOpts.EnableTLS {
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
			if err == http.ErrServerClosed {
				klog.V(1).Info("Server successfully stopped")
			} else {
				klog.Fatalf("EventsRunnerAPI: ListenAndServe() failed: %v", err)
			}
		}
	}
	go startServer(&wg, erapi.server)
	go startServer(&wg, erapi.healthzServer)
	wg.Wait()
}

// Stop gracefully shuts down the server and its components.
func (erapi *EventsRunnerAPI) Stop(ctx context.Context) error {
	klog.V(1).Info("Stopping eventsrunner-api healthz server")
	timedCtx, cancelFunc := context.WithTimeout(ctx, 5*time.Minute)
	defer cancelFunc()
	if err := erapi.server.Shutdown(timedCtx); err != nil {
		return err
	}
	klog.V(1).Info("Stopping eventsrunner-api server")
	if err := erapi.healthzServer.Shutdown(timedCtx); err != nil {
		return err
	}
	return nil
}

// StopOnSignal gracefully shuts down the server and its components when
// SIGTERM or SIGINT is received.
func (erapi *EventsRunnerAPI) StopOnSignal() {
	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	<-signalChan
	erapi.Stop(context.Background())
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
		return validationError{Msg: "eventType must be one of added, updated, or deleted"}
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

/*
eventPostHandler handles POST requests to the /api/v1/events endpoint.
It parses the request and creates an event resource under the configured namespace.
Objects in the event data array will be parsed as string representations of JSON and will
be added into event resource.

If the request is invalid, it returns a 400 Bad Request response.

If the request is valid, it returns a 200 Okay response with the resource name.

If the api is throttled, it returns a 429 Too Many Requests response. This could happen due to
the kube-apiserver rate limiting which would prevent the eventsrunner-api to create event crds.
Client should handle retries for 429.
*/
// TODO: Add tests
// TODO: Benchmark performance
func (erapi *EventsRunnerAPI) eventPostHandler(w http.ResponseWriter, r *http.Request) {
	// Only allow POST requests
	if r.Method != http.MethodPost {
		handleEventPostResponse(w, eventPostResponseBody{
			Error: "Method not allowed",
		}, http.StatusMethodNotAllowed)
		return
	}
	// Parse and validate request body
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
		Status: erAPI.EventStatus{
			State:   erAPI.PENDING,
			Retries: 0,
		},
	}
	event, err := erapi.eventsRunnerClient.Events(erapi.namespace).Create(r.Context(), eventObj, metav1.CreateOptions{})
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
