package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	_ "github.com/lib/pq"
	"github.com/mark3labs/mcp-go/server"
)

// PostgresServer wraps a *sql.DB and exposes read-mostly tools.
type PostgresServer struct {
	mcp          *server.MCPServer
	db           *sql.DB
	allowWrite   bool
	maxRows      int
	queryTimeout int
}

func (p *PostgresServer) MCP() *server.MCPServer { return p.mcp }
func (p *PostgresServer) DB() *sql.DB             { return p.db }
func (p *PostgresServer) AllowWrite() bool        { return p.allowWrite }
func (p *PostgresServer) MaxRows() int            { return p.maxRows }
func (p *PostgresServer) QueryTimeout() int       { return p.queryTimeout }

// Close releases the underlying DB pool.
func (p *PostgresServer) Close() error {
	if p.db != nil {
		return p.db.Close()
	}
	return nil
}

// NewPostgresServer opens the DSN given by POSTGRES_DSN, pings it, and wires
// the MCP server. PG_ALLOW_WRITE, PG_MAX_ROWS, PG_QUERY_TIMEOUT are optional.
func NewPostgresServer() (*PostgresServer, error) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		return nil, fmt.Errorf("POSTGRES_DSN environment variable is required")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("sql.Open: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	db.SetConnMaxLifetime(time.Hour)

	maxRows := 500
	if v := os.Getenv("PG_MAX_ROWS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			_ = db.Close()
			return nil, fmt.Errorf("invalid PG_MAX_ROWS %q", v)
		}
		maxRows = n
	}
	timeout := 30
	if v := os.Getenv("PG_QUERY_TIMEOUT"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			_ = db.Close()
			return nil, fmt.Errorf("invalid PG_QUERY_TIMEOUT %q", v)
		}
		timeout = n
	}

	mcp := server.NewMCPServer("postgres", "1.0.0", server.WithToolCapabilities(true))
	p := &PostgresServer{
		mcp:          mcp,
		db:           db,
		allowWrite:   os.Getenv("PG_ALLOW_WRITE") == "true",
		maxRows:      maxRows,
		queryTimeout: timeout,
	}
	p.registerTools()
	return p, nil
}

// queryCtx returns a context bounded by PG_QUERY_TIMEOUT, derived from the
// MCP call context so server-side cancellation still propagates.
func (p *PostgresServer) queryCtx(parent context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(parent, time.Duration(p.queryTimeout)*time.Second)
}
