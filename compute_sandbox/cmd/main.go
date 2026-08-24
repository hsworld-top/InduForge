package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/indu-forge/compute_sandbox/internal/sandbox"
)

func main() {
	config, err := sandbox.LoadConfig()
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: config.Addr, Handler: sandbox.NewHandler(config)}
	log.Printf("compute_sandbox listening on %s", config.Addr)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		if !strings.Contains(err.Error(), "closed") {
			log.New(os.Stderr, "", log.LstdFlags).Fatal(err)
		}
	}
}
