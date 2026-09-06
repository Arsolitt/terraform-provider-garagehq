package main

import (
	"testing"
)

func websiteBlock(enabled bool, index, errDoc string) []interface{} {
	return []interface{}{
		map[string]interface{}{
			"enabled":        enabled,
			"index_document": index,
			"error_document": errDoc,
		},
	}
}

func TestBuildWebsiteAccess(t *testing.T) {
	t.Run("absent block disables access", func(t *testing.T) {
		wa, err := buildWebsiteAccess(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.GetEnabled() {
			t.Error("expected enabled=false")
		}
		if wa.HasIndexDocument() || wa.HasErrorDocument() {
			t.Error("expected no documents when block is absent")
		}
	})

	t.Run("empty list disables access", func(t *testing.T) {
		wa, err := buildWebsiteAccess([]interface{}{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.GetEnabled() {
			t.Error("expected enabled=false")
		}
	})

	t.Run("enabled without index_document errors", func(t *testing.T) {
		_, err := buildWebsiteAccess(websiteBlock(true, "", ""))
		if err == nil {
			t.Fatal("expected error when enabled with no index_document")
		}
	})

	t.Run("enabled with index and error documents", func(t *testing.T) {
		wa, err := buildWebsiteAccess(websiteBlock(true, "index.html", "error.html"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !wa.GetEnabled() {
			t.Error("expected enabled=true")
		}
		if wa.GetIndexDocument() != "index.html" {
			t.Errorf("index_document = %q, want index.html", wa.GetIndexDocument())
		}
		if wa.GetErrorDocument() != "error.html" {
			t.Errorf("error_document = %q, want error.html", wa.GetErrorDocument())
		}
	})

	t.Run("enabled with index only", func(t *testing.T) {
		wa, err := buildWebsiteAccess(websiteBlock(true, "index.html", ""))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.HasErrorDocument() {
			t.Error("expected no error_document when not set")
		}
	})

	t.Run("disabled drops leftover documents", func(t *testing.T) {
		wa, err := buildWebsiteAccess(websiteBlock(false, "index.html", "error.html"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if wa.GetEnabled() {
			t.Error("expected enabled=false")
		}
		if wa.HasIndexDocument() || wa.HasErrorDocument() {
			t.Error("expected documents dropped when disabled (Garage rejects them)")
		}
	})
}
