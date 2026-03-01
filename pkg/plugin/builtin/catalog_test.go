package builtin

import (
	"slices"
	"testing"
)

func TestCatalogContainsPolicyDemo(t *testing.T) {
	catalog := Catalog()
	factory, ok := catalog["policy-demo"]
	if !ok {
		t.Fatalf("Catalog() missing %q plugin", "policy-demo")
	}
	if factory == nil {
		t.Fatalf("Catalog()[%q] factory is nil", "policy-demo")
	}
	if got := factory(); got == nil {
		t.Fatalf("Catalog()[%q]() returned nil plugin", "policy-demo")
	}
}

func TestNamesSorted(t *testing.T) {
	first := Names()
	second := Names()

	if !slices.IsSorted(first) {
		t.Fatalf("Names() is not sorted: %v", first)
	}
	if !slices.Equal(first, second) {
		t.Fatalf("Names() is not deterministic across calls: %v vs %v", first, second)
	}
}
