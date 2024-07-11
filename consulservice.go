package consulservice

import (
	"fmt"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/go-uuid"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// Service represents a Consul service
type Service struct {
	client *api.Client
	ID     string
}

// NewService creates and registers a new service with Consul
func NewService(serviceName string, serviceTags []string, servicePort int) (*Service, error) {
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to create Consul client: %w", err)
	}

	serviceID, err := uuid.GenerateUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate service ID: %w", err)
	}

	serviceRegistration := &api.AgentServiceRegistration{
		ID:      serviceID,   // Unique ID for the service
		Name:    serviceName, // Name of the service
		Tags:    serviceTags, // Tags for the service
		Address: "127.0.0.1", // Address of the service
		Port:    servicePort, // Port of the service
		Check: &api.AgentServiceCheck{
			HTTP:     fmt.Sprintf("http://127.0.0.1:%d/health", servicePort), // Health check endpoint
			Interval: "10s",                                                  // Interval for health checks
			Timeout:  "1s",                                                   // Timeout for health checks
		},
	}

	// Register the service with Consul
	err = client.Agent().ServiceRegister(serviceRegistration)
	if err != nil {
		return nil, fmt.Errorf("failed to register service: %w", err)
	}

	fmt.Println("Service registered successfully.")

	// Start the health check endpoint
	http.HandleFunc("/health", healthCheckHandler)
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", servicePort), nil))
	}()

	s := &Service{
		client: client,
		ID:     serviceID,
	}

	go s.handleDeregistrationOnExit()

	return s, nil
}

// healthCheckHandler is the HTTP handler for the health check endpoint
func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// handleDeregistrationOnExit handles deregistration of the service when the program exits
func (s *Service) handleDeregistrationOnExit() {
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	<-signalChan

	fmt.Println("Received termination signal, deregistering service...")

	err := s.client.Agent().ServiceDeregister(s.ID)
	if err != nil {
		log.Fatalf("Failed to deregister service: %v", err)
	}

	fmt.Println("Service deregistered successfully.")
}
