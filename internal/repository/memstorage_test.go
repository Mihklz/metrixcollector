package repository

import (
	"testing"
)

func TestMemStorage_UpdateGauge(t *testing.T) {
	s := NewMemStorage()
	err := s.Update("gauge", "Alloc", "123.45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := s.Gauges["Alloc"]; got != 123.45 {
		t.Errorf("expected 123.45, got %v", got)
	}
}

func TestMemStorage_UpdateCounter(t *testing.T) {
	s := NewMemStorage()
	_ = s.Update("counter", "PollCount", "5")
	_ = s.Update("counter", "PollCount", "3")

	if got := s.Counters["PollCount"]; got != 8 {
		t.Errorf("expected 8, got %v", got)
	}
}

func TestMemStorage_UpdateInvalidType(t *testing.T) {
	s := NewMemStorage()
	err := s.Update("invalid", "Some", "100")
	if err == nil {
		t.Error("expected error for unsupported type, got nil")
	}
}

func TestMemStorage_UpdateInvalidValue(t *testing.T) {
	s := NewMemStorage()
	err := s.Update("gauge", "Alloc", "abc")
	if err == nil {
		t.Error("expected error for invalid float, got nil")
	}
}
