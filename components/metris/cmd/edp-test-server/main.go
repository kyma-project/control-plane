package main

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alecthomas/kong"
	"github.com/xeipuuv/gojsonschema"
)

var (
	schemaLoader gojsonschema.JSONLoader
)

type cli struct {
	ListenAddr string `kong:"help='Address and port the HTTP endpoints will bind to.',default=:5678"`
	EDPSchema  string `kong:"help='Path to the schema file used to validate requests in EDP',type=existingfile,required,default=edp_data_schema.json"`
}

func main() {
	app := cli{}
	_ = kong.Parse(&app,
		kong.Name("edp test server"),
		kong.UsageOnError(),
		kong.ConfigureHelp(kong.HelpOptions{Compact: true}),
	)

	// Flag gets printed as a page
	mux := http.NewServeMux()
	mux.HandleFunc("/", httpEcho())

	server := &http.Server{
		Addr:    app.ListenAddr,
		Handler: mux,
	}

	schemaLoader = gojsonschema.NewReferenceLoader(fmt.Sprintf("file://%s", app.EDPSchema))

	log.Printf("[INFO] server is listening on %s\n", app.ListenAddr)

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("[ERR] server exited with: %s", err)
	}

	log.Printf("[INFO] received interrupt, shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[ERR] failed to shutdown server: %s", err)
	}
}

func httpEcho() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r.Body); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := r.Body.Close(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		buf.Write([]byte("\n"))
		documentLoader := gojsonschema.NewStringLoader(buf.String())

		result, err := gojsonschema.Validate(schemaLoader, documentLoader)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(err.Error()))
			return
		}

		buf.WriteTo(os.Stdout)
		if result.Valid() {
			w.WriteHeader(http.StatusCreated)
			fmt.Print("Valid request")
		} else {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Print("Invalid request")
			for _, desc := range result.Errors() {
				w.Write([]byte(fmt.Sprintf("- %s\n", desc)))
				fmt.Printf("- %s\n", desc)
			}
		}
	}
}
