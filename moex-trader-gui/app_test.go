package main

import "testing"

func TestAffordableQuantity_ReturnsZeroWhenInsufficientCash(t *testing.T) {
	a := &App{}
	qty := a.affordableQuantity(1000, 2000, 10, false, "TEST")
	if qty != 0 {
		t.Fatalf("expected 0 quantity, got %d", qty)
	}
}

func TestAffordableQuantity_CapsByCash(t *testing.T) {
	a := &App{}
	qty := a.affordableQuantity(10000, 1000, 20, false, "TEST")
	if qty <= 0 || qty >= 20 {
		t.Fatalf("expected capped positive quantity (<20), got %d", qty)
	}
}

func TestFloorQuantityByLot(t *testing.T) {
	tests := []struct {
		name     string
		quantity int
		lot      int
		want     int
	}{
		{name: "exact lot", quantity: 30, lot: 10, want: 30},
		{name: "round down", quantity: 27, lot: 10, want: 20},
		{name: "below lot", quantity: 7, lot: 10, want: 0},
		{name: "lot one", quantity: 7, lot: 1, want: 7},
		{name: "invalid lot", quantity: 7, lot: 0, want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := floorQuantityByLot(tt.quantity, tt.lot)
			if got != tt.want {
				t.Fatalf("floorQuantityByLot(%d,%d)=%d, want %d", tt.quantity, tt.lot, got, tt.want)
			}
		})
	}
}
