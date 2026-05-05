package tests

import (
	"errors"
	"testing"

	"easywired/internal/ipam"
	"easywired/internal/model"
)

func TestIPAMAllocatesFromLowToHigh(t *testing.T) {
	cfg := &model.NodeConfig{Interface: model.WGInterface{Address: "10.0.0.1/24"}}
	allocator := ipam.NewAllocator()
	lease, err := allocator.Allocate(cfg, "nodeB", "bbb")
	if err != nil {
		t.Fatal(err)
	}
	if lease.Address != "10.0.0.2/32" {
		t.Fatalf("expected 10.0.0.2/32, got %s", lease.Address)
	}
	lease, err = allocator.Allocate(cfg, "nodeC", "ccc")
	if err != nil {
		t.Fatal(err)
	}
	if lease.Address != "10.0.0.3/32" {
		t.Fatalf("expected 10.0.0.3/32, got %s", lease.Address)
	}
}

func TestIPAMIdempotentByPublicKey(t *testing.T) {
	cfg := &model.NodeConfig{Interface: model.WGInterface{Address: "10.0.0.1/24"}}
	allocator := ipam.NewAllocator()
	first, err := allocator.Allocate(cfg, "nodeB", "bbb")
	if err != nil {
		t.Fatal(err)
	}
	second, err := allocator.Allocate(cfg, "nodeB-renamed", "bbb")
	if err != nil {
		t.Fatal(err)
	}
	if first.Address != second.Address || len(cfg.Leases) != 1 {
		t.Fatalf("expected idempotent allocation, got %v %v leases=%d", first, second, len(cfg.Leases))
	}
}

func TestIPAM32PoolNotAllocatable(t *testing.T) {
	cfg := &model.NodeConfig{Interface: model.WGInterface{Address: "10.0.0.1/32"}}
	_, err := ipam.NewAllocator().Allocate(cfg, "nodeB", "bbb")
	if !errors.Is(err, ipam.ErrNoAddressPool) {
		t.Fatalf("expected ErrNoAddressPool, got %v", err)
	}
}
