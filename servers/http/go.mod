module github.com/dimasd-angga/go-mcp-servers/servers/http

go 1.23

require (
	github.com/dimasd-angga/go-mcp-servers/shared v0.0.0
	github.com/mark3labs/mcp-go v0.31.0
	github.com/rs/zerolog v1.33.0
)

replace github.com/dimasd-angga/go-mcp-servers/shared => ../../shared
