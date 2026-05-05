package api

import (
	"net/http"
	"sync"

	"easywired/internal/auth"
	"easywired/internal/backend"
	"easywired/internal/ipam"
	"easywired/internal/store"
)

type Server struct {
	store      *store.Store
	backend    backend.Backend
	deviceName string
	listenAddr string
	outputPath string
	allocator  *ipam.Allocator
	mu         sync.Mutex
}

type Options struct {
	Store      *store.Store
	Backend    backend.Backend
	DeviceName string
	ListenAddr string
	OutputPath string
}

func NewServer(opts Options) *Server {
	return &Server{
		store:      opts.Store,
		backend:    opts.Backend,
		deviceName: opts.DeviceName,
		listenAddr: opts.ListenAddr,
		outputPath: opts.OutputPath,
		allocator:  ipam.NewAllocator(),
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/connect", s.handleConnect)
	mux.HandleFunc("/disconnect", s.handleDisconnect)
	mux.HandleFunc("/peers", s.handlePeers)
	return auth.Passthrough(mux)
}

func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.listenAddr, s.Handler())
}
