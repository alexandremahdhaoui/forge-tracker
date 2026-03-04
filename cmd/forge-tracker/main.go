// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/alexandremahdhaoui/forge-tracker/internal/adapter/markdown"
	"github.com/alexandremahdhaoui/forge-tracker/internal/controller"
	rest "github.com/alexandremahdhaoui/forge-tracker/internal/driver/rest"
)

func main() {
	storagePath := flag.String("storage-path", "", "path to tracker storage directory (required)")
	addr := flag.String("addr", ":8081", "HTTP listen address")
	flag.Parse()

	if *storagePath == "" {
		log.Fatal("--storage-path is required")
	}

	// Create store and build label index.
	store := markdown.NewStore(*storagePath)
	if err := store.BuildIndex(); err != nil {
		log.Fatalf("build label index: %v", err)
	}

	// Create controllers.
	ticketSvc := controller.NewTicketService(store.TicketStore(), store.GraphStore())
	graphSvc := controller.NewGraphService(store.GraphStore(), store.TicketStore())
	tsSvc := controller.NewTrackingSetService(store.TrackingSetStore())
	planSvc := controller.NewPlanService(store.PlanStore(), store.GraphStore())
	mpSvc := controller.NewMetaPlanService(store.MetaPlanStore())

	// Create REST API handler and register routes.
	handler := rest.NewAPIHandler(ticketSvc, graphSvc, tsSvc, planSvc, mpSvc)
	mux := http.NewServeMux()

	// Health check endpoint.
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	si := rest.NewStrictHandler(handler, nil)
	rest.HandlerFromMux(si, mux)

	// Start server with graceful shutdown.
	srv := &http.Server{Addr: *addr, Handler: corsMiddleware(mux)}

	go func() {
		log.Printf("forge-tracker listening on %s", *addr)
		if err := srv.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("shutting down...")
	_ = srv.Close()
}

// corsMiddleware wraps an http.Handler with permissive CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
