package server

import (
	"sync"

	"EasyWired/wg"
)

type Store struct {
	mu    sync.RWMutex
	nodes map[string]NodeRecord
}

func NewStore() *Store {
	return &Store{
		nodes: make(map[string]NodeRecord),
	}
}

func (s *Store) Get(nodeID string) (NodeRecord, bool) {
	s.mu.RLock()
	record, ok := s.nodes[nodeID]
	s.mu.RUnlock()
	if !ok {
		return NodeRecord{}, false
	}

	return cloneNodeRecord(record), true
}

func (s *Store) Set(nodeID string, record NodeRecord) {
	s.mu.Lock()
	s.nodes[nodeID] = cloneNodeRecord(record)
	s.mu.Unlock()
}

func (s *Store) All() []NodeRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]NodeRecord, 0, len(s.nodes))
	for _, record := range s.nodes {
		out = append(out, cloneNodeRecord(record))
	}

	return out
}

func cloneNodeRecord(record NodeRecord) NodeRecord {
	cloned := record
	cloned.AllowedCIDRs = append([]string(nil), record.AllowedCIDRs...)
	cloned.SuggestedDNS = append([]string(nil), record.SuggestedDNS...)
	cloned.Peers = append([]wg.PeerEntry(nil), record.Peers...)
	return cloned
}
