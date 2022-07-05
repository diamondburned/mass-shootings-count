package main

import (
	"context"
	"flag"
	"net/http"
	"os"
	"os/signal"

	"github.com/diamondburned/listener"
	"github.com/diamondburned/mass-shootings-count/cmd/mass-shootings-count/frontend"
	"github.com/diamondburned/mass-shootings-count/cmd/mass-shootings-count/frontend/index"
	"github.com/diamondburned/mass-shootings-count/gva"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func main() {
	var addr string
	var tmpl string

	flag.StringVar(&addr, "addr", "localhost:8081", "http listening address")
	flag.StringVar(&tmpl, "tmpl", "", "optional path to override templates")
	flag.Parse()

	if tmpl != "" {
		frontend.OverrideTmpl(tmpl)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Mount("/static", http.StripPrefix("/static", frontend.StaticHandler()))
	r.Mount("/", index.Mount(gva.NewScraper()))

	listener.MustHTTPListenAndServeCtx(ctx, &http.Server{
		Addr:    addr,
		Handler: r,
	})
}
