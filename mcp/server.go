package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"sync"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var nameRe = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,64}$`)

type CallFunc func(ctx context.Context, t Tool, args json.RawMessage) (any, error)

// Middleware membungkus pemanggilan tool (logging, metrics, rate limit, dll.).
type Middleware func(next CallFunc) CallFunc

type Option func(*Server)

func WithPermission(p Permission) Option     { return func(s *Server) { s.perm = p } }
func WithMiddleware(mw ...Middleware) Option { return func(s *Server) { s.mws = append(s.mws, mw...) } }

type Server struct {
	name, version string
	perm          Permission
	mws           []Middleware

	mu    sync.RWMutex
	tools map[string]Tool
	order []string
}

func New(name, version string, opts ...Option) *Server {
	s := &Server{name: name, version: version, perm: AllowAll(), tools: map[string]Tool{}}
	for _, o := range opts {
		o(s)
	}
	return s
}

func (s *Server) Add(tools ...Tool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range tools {
		switch {
		case !nameRe.MatchString(t.Name):
			return fmt.Errorf("invalid tool name %q (letters, digits, _ and -, max 64)", t.Name)
		case t.call == nil:
			return fmt.Errorf("tool %q: build it with mcptools.Func or mcptools.Raw", t.Name)
		}
		if _, dup := s.tools[t.Name]; dup {
			return fmt.Errorf("tool %q already registered", t.Name)
		}
		s.tools[t.Name] = t
		s.order = append(s.order, t.Name)
	}
	return nil
}

// Call menjalankan tool tanpa lewat MCP (agent loop sendiri, test, dll.).
// Urutan: middleware -> permission -> timeout -> fungsi. Panic diubah jadi error.
func (s *Server) Call(ctx context.Context, name string, args json.RawMessage) (out any, err error) {
	s.mu.RLock()
	t, ok := s.tools[name]
	s.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown tool %q", name)
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("tool %q panicked: %v", name, r)
		}
	}()

	var call CallFunc = func(ctx context.Context, t Tool, args json.RawMessage) (any, error) {
		if err := s.perm.Allow(ctx, Action{Tool: t.Name, ReadOnly: t.ReadOnly, Description: t.describeArgs(args)}); err != nil {
			return nil, err
		}
		if t.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, t.Timeout)
			defer cancel()
		}
		return t.call(ctx, args)
	}
	for i := len(s.mws) - 1; i >= 0; i-- {
		call = s.mws[i](call)
	}
	return call(ctx, t, args)
}

// Tools mengembalikan tool dalam format mcp-go.
func (s *Server) Tools() []server.ServerTool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]server.ServerTool, 0, len(s.order))
	for _, name := range s.order {
		t := s.tools[name]
		mt := mcp.NewToolWithRawSchema(t.Name, t.Description, t.Schema)
		ro := t.ReadOnly
		mt.Annotations.ReadOnlyHint = &ro
		out = append(out, server.ServerTool{Tool: mt, Handler: s.handler(name)})
	}
	return out
}

// Register menempelkan semua tool ke MCPServer milikmu sendiri.
func (s *Server) Register(m *server.MCPServer) { m.AddTools(s.Tools()...) }

func (s *Server) MCPServer() *server.MCPServer {
	m := server.NewMCPServer(s.name, s.version, server.WithToolCapabilities(true))
	s.Register(m)
	return m
}

func (s *Server) ServeStdio() error { return server.ServeStdio(s.MCPServer()) }

func (s *Server) handler(name string) server.ToolHandlerFunc {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		raw, err := json.Marshal(req.Params.Arguments)
		if err != nil {
			return mcp.NewToolResultError("invalid arguments: " + err.Error()), nil
		}
		out, err := s.Call(ctx, name, raw)
		if err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}
		return toResult(out), nil
	}
}

// toResult: string -> teks, nil -> "ok", *mcp.CallToolResult -> apa adanya (gambar, dll.), lainnya -> JSON.
func toResult(out any) *mcp.CallToolResult {
	switch v := out.(type) {
	case nil:
		return mcp.NewToolResultText("ok")
	case string:
		return mcp.NewToolResultText(v)
	case *mcp.CallToolResult:
		return v
	}
	b, err := json.Marshal(out)
	if err != nil {
		return mcp.NewToolResultError(err.Error())
	}
	return mcp.NewToolResultText(string(b))
}
