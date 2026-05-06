package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"easywired/internal/config"
	"easywired/internal/model"
)

type Store struct {
	path string
	mu   sync.Mutex
	cfg  *model.NodeConfig
}

func LoadConfig(path string) (*model.NodeConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg model.NodeConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, err
	}
	if _, err := ensureConfigDefaults(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func LoadOrCreateConfig(path string) (*model.NodeConfig, error) {
	cfg, err := LoadConfig(path)
	if err == nil {
		return cfg, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}
	cfg = &model.NodeConfig{Peers: []model.WGPeer{}}
	if _, err := ensureConfigDefaults(cfg); err != nil {
		return nil, err
	}
	if err := SaveConfig(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ensureConfigDefaults(cfg *model.NodeConfig) (bool, error) {
	changed, err := config.EnsureNodeID(cfg, config.GenerateNodeID)
	if err != nil {
		return false, err
	}
	keysChanged, err := config.EnsureInterfaceKeys(cfg)
	if err != nil {
		return false, err
	}
	return changed || keysChanged, nil
}

func SaveConfig(path string, cfg *model.NodeConfig) error {
	if cfg == nil {
		return errors.New("nil config")
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(b); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, path)
}

func Open(path string) (*Store, error) {
	cfg, err := LoadConfig(path)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, cfg: cfg}, nil
}

func OpenOrCreate(path string) (*Store, error) {
	cfg, err := LoadOrCreateConfig(path)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, cfg: cfg}, nil
}

func New(path string, cfg *model.NodeConfig) *Store {
	return &Store{path: path, cfg: cfg}
}

func (s *Store) Config() *model.NodeConfig {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.cfg
}

func (s *Store) Save() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return SaveConfig(s.path, s.cfg)
}

func (s *Store) UpsertPeer(peer model.WGPeer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.cfg.Peers {
		if s.cfg.Peers[i].PublicKey == peer.PublicKey {
			s.cfg.Peers[i] = peer
			return nil
		}
	}
	s.cfg.Peers = append(s.cfg.Peers, peer)
	return nil
}

func (s *Store) RemovePeer(publicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	peers := s.cfg.Peers[:0]
	for _, peer := range s.cfg.Peers {
		if peer.PublicKey != publicKey {
			peers = append(peers, peer)
		}
	}
	s.cfg.Peers = peers
	return nil
}

func (s *Store) UpsertLease(lease model.Lease) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.cfg.Leases {
		if s.cfg.Leases[i].PublicKey == lease.PublicKey || lease.NodeID != "" && s.cfg.Leases[i].NodeID == lease.NodeID {
			s.cfg.Leases[i] = lease
			return nil
		}
	}
	s.cfg.Leases = append(s.cfg.Leases, lease)
	return nil
}

func (s *Store) RemoveLease(publicKey string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	leases := s.cfg.Leases[:0]
	for _, lease := range s.cfg.Leases {
		if lease.PublicKey != publicKey {
			leases = append(leases, lease)
		}
	}
	s.cfg.Leases = leases
	return nil
}

func (s *Store) FindLeaseByPublicKey(publicKey string) (*model.Lease, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, lease := range s.cfg.Leases {
		if lease.PublicKey == publicKey {
			copy := lease
			return &copy, true
		}
	}
	return nil, false
}

func (s *Store) FindPeerByPublicKey(publicKey string) (*model.WGPeer, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, peer := range s.cfg.Peers {
		if peer.PublicKey == publicKey {
			copy := peer
			return &copy, true
		}
	}
	return nil, false
}
