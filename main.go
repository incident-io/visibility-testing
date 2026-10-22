// Command deploy-status serves the deploy order described in README.md and reports whether
// each environment's health endpoint is up, so the agents watching the pipeline can poll it.
package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"
)

// environment is one stage of the deploy order, with the URL its health check lives at.
type environment struct {
	Name      string `json:"name"`
	HealthURL string `json:"health_url"`
}

// deployOrder mirrors the order in README.md: alpha, then beta, then theta (production).
var deployOrder = []environment{
	{Name: "alpha", HealthURL: envOr("ALPHA_HEALTH_URL", "http://localhost:9001/healthz")},
	{Name: "beta", HealthURL: envOr("BETA_HEALTH_URL", "http://localhost:9002/healthz")},
	{Name: "theta", HealthURL: envOr("THETA_HEALTH_URL", "http://localhost:9003/healthz")},
}

type environmentStatus struct {
	environment
	Healthy bool   `json:"healthy"`
	Error   string `json:"error,omitempty"`
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("GET /deploy-order", handleDeployOrder)
	mux.HandleFunc("GET /deploy-status", handleDeployStatus)

	addr := envOr("ADDR", ":8080")
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("deploy-status listening on %s", addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}

// handleDeployOrder returns the environments in the order a deploy should reach them.
func handleDeployOrder(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, deployOrder)
}

// handleDeployStatus checks each environment's health endpoint in deploy order. A deploy
// should not move to the next environment until the previous one reports healthy.
func handleDeployStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	client := &http.Client{Timeout: 3 * time.Second}
	statuses := make([]environmentStatus, 0, len(deployOrder))
	for _, env := range deployOrder {
		status := environmentStatus{environment: env}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, env.HealthURL, nil)
		if err != nil {
			status.Error = err.Error()
			statuses = append(statuses, status)
			continue
		}

		resp, err := client.Do(req)
		if err != nil {
			status.Error = err.Error()
		} else {
			status.Healthy = resp.StatusCode == http.StatusOK
			resp.Body.Close()
		}
		statuses = append(statuses, status)
	}

	writeJSON(w, http.StatusOK, statuses)
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		log.Printf("encoding response: %v", err)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
