package main

import (
	"flag"
	"fmt"
	"os"

	sharedlog "github.com/dimasd-angga/go-mcp-servers/shared/logger"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	transport := flag.String("transport", "stdio", "Transport: stdio or sse")
	port := flag.Int("port", 8002, "Port for SSE transport")
	flag.Parse()

	log := sharedlog.Init("postgres")

	p, err := NewPostgresServer()
	if err != nil {
		log.Fatal().Err(err).Msg("init failed")
	}
	defer p.Close()

	log.Info().
		Bool("allow_write", p.AllowWrite()).
		Int("max_rows", p.MaxRows()).
		Int("query_timeout_s", p.QueryTimeout()).
		Str("transport", *transport).
		Msg("postgres MCP server starting")

	switch *transport {
	case "stdio":
		if err := server.ServeStdio(p.MCP()); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	case "sse":
		sse := server.NewSSEServer(p.MCP(),
			server.WithBaseURL(fmt.Sprintf("http://localhost:%d", *port)),
		)
		log.Info().Int("port", *port).Msg("listening on SSE")
		if err := sse.Start(fmt.Sprintf(":%d", *port)); err != nil {
			log.Fatal().Err(err).Msg("SSE server failed")
		}
	default:
		log.Fatal().Str("transport", *transport).Msg("unknown transport")
	}
}
