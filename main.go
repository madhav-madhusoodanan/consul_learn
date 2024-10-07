package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/connect"
)

func main() {
	// Create a Consul client
	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatalf("Failed to create Consul client: %v", err)
	}

	// Start a service
	// service1 := startService("service1", ":8080", client)
	// defer service1.Close()

	// Start a client that communicates with the service
	startClient("service2", "service1", client)
}

func startService(name string, addr string, client *api.Client) *connect.Service {
	// Create a Consul service
	service := &api.AgentServiceRegistration{
		Name: name,
		Port: 8080,
		Address: "127.0.0.1",
		Check: &api.AgentServiceCheck{
			HTTP:     "http://localhost:8080",
			Interval: "5s",
			TLSSkipVerify: true,
		},
	}
	if err := client.Agent().ServiceRegister(service); err != nil {
		log.Fatalf("Failed to register service: %v", err)
	}

	// Create a Connect-enabled service
	svc, err := connect.NewService(name, client)
	if err != nil {
		log.Fatalf("Failed to create Connect service: %v", err)
	}

	// Start an HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from %s!", name)
	})

	// Serve using the Connect service
	log.Fatal(http.ListenAndServe(addr, nil))

	return svc
}

func startClient(name string, targetService string, client *api.Client) {
	// Create a Consul service for the client
	service := &api.AgentServiceRegistration{
		Name: name,
	}
	if err := client.Agent().ServiceRegister(service); err != nil {
		log.Fatalf("Failed to register client service: %v", err)
	}

	// Create a Connect-enabled client
	svc, err := connect.NewService(name, client)
	if err != nil {
		log.Fatalf("Failed to create Connect client: %v", err)
	}

	// Create an HTTP client using the Connect service
	httpClient := svc.HTTPClient()

	// Use service discovery to find the target service
	services, _, err := client.Catalog().Service(targetService, "", nil)
	if err != nil || len(services) == 0 {
		log.Fatalf("Failed to discover service: %v", err)
	}

	// Make a request to the target service
	resp, err := httpClient.Get(fmt.Sprintf("https://%s.service.consul", targetService))
	if err != nil {
		log.Fatalf("Failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Response from %s: %s\n", targetService, string(body))

	if err := client.Agent().ServiceDeregister(name); err != nil {
		log.Fatalf("Failed to register client service: %v", err)
	}
}
