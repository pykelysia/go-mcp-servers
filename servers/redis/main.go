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
	port := flag.Int("port", 8004, "Port for SSE transport")
	flag.Parse()

	log := sharedlog.Init("redis")

	r, err := NewRedisServer()
	if err != nil {
		log.Fatal().Err(err).Msg("init failed")
	}
	defer r.Close()

	log.Info().
		Str("prefix", r.Prefix()).
		Str("transport", *transport).
		Msg("redis MCP server starting")

	switch *transport {
	case "stdio":
		if err := server.ServeStdio(r.MCP()); err != nil {
			fmt.Fprintf(os.Stderr, "server error: %v\n", err)
			os.Exit(1)
		}
	case "sse":
		sse := server.NewSSEServer(r.MCP(),
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
