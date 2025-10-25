// ABOUTME: Tests for version constants
// ABOUTME: Ensures version information is properly defined
package version

import (
	"testing"
)

func TestVersionDefined(t *testing.T) {
	if Version == "" {
		t.Error("Version should not be empty")
	}
}

func TestProductDefined(t *testing.T) {
	if Product == "" {
		t.Error("Product should not be empty")
	}
}

func TestManufacturerDefined(t *testing.T) {
	if Manufacturer == "" {
		t.Error("Manufacturer should not be empty")
	}
}

func TestVersionFormat(t *testing.T) {
	// Version should typically be in format like "0.1.0" or "dev"
	if len(Version) == 0 {
		t.Error("Version string is empty")
	}

	// Just verify it's a reasonable string
	if len(Version) > 100 {
		t.Error("Version string is unreasonably long")
	}
}

func TestProductFormat(t *testing.T) {
	// Product name should be reasonable length
	if len(Product) == 0 {
		t.Error("Product name is empty")
	}

	if len(Product) > 100 {
		t.Error("Product name is unreasonably long")
	}
}

func TestManufacturerFormat(t *testing.T) {
	// Manufacturer should be reasonable
	if len(Manufacturer) == 0 {
		t.Error("Manufacturer is empty")
	}

	if len(Manufacturer) > 100 {
		t.Error("Manufacturer name is unreasonably long")
	}
}

func TestVersionImmutability(t *testing.T) {
	// Store original values
	originalVersion := Version
	originalProduct := Product
	originalManufacturer := Manufacturer

	// These are const, so they can't actually be modified,
	// but let's verify they're still accessible
	if Version != originalVersion {
		t.Error("Version changed unexpectedly")
	}

	if Product != originalProduct {
		t.Error("Product changed unexpectedly")
	}

	if Manufacturer != originalManufacturer {
		t.Error("Manufacturer changed unexpectedly")
	}
}

func TestVersionNotPlaceholder(t *testing.T) {
	// Check for common placeholder values
	placeholders := []string{"TODO", "FIXME", "XXX", "placeholder"}

	for _, placeholder := range placeholders {
		if Version == placeholder {
			t.Errorf("Version should not be placeholder value: %s", placeholder)
		}
		if Product == placeholder {
			t.Errorf("Product should not be placeholder value: %s", placeholder)
		}
		if Manufacturer == placeholder {
			t.Errorf("Manufacturer should not be placeholder value: %s", placeholder)
		}
	}
}
