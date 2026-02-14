package server

import (
	"encoding/json"
	"testing"
	"testing/fstest"
)

func TestPWAManifestBytesUsesDevNameOverrideInDevMode(t *testing.T) {
	t.Setenv(devPWANameEnvVar, "SimpleDMS Local")

	pwaManifestHandler := NewPWAManifestHandler(testManifestFS(), true)
	manifestBytes, err := pwaManifestHandler.manifestBytes()
	if err != nil {
		t.Fatalf("pwa manifest bytes: %v", err)
	}

	manifestData := parseManifestData(t, manifestBytes)
	if manifestData.Name != "SimpleDMS Local" {
		t.Fatalf("expected name %q, got %q", "SimpleDMS Local", manifestData.Name)
	}
	if manifestData.ShortName != "SimpleDMS Local" {
		t.Fatalf("expected short_name %q, got %q", "SimpleDMS Local", manifestData.ShortName)
	}
}

func TestPWAManifestBytesDoesNotOverrideOutsideDevMode(t *testing.T) {
	t.Setenv(devPWANameEnvVar, "SimpleDMS Local")

	pwaManifestHandler := NewPWAManifestHandler(testManifestFS(), false)
	manifestBytes, err := pwaManifestHandler.manifestBytes()
	if err != nil {
		t.Fatalf("pwa manifest bytes: %v", err)
	}

	manifestData := parseManifestData(t, manifestBytes)
	if manifestData.Name != "SimpleDMS" {
		t.Fatalf("expected name %q, got %q", "SimpleDMS", manifestData.Name)
	}
	if manifestData.ShortName != "SimpleDMS" {
		t.Fatalf("expected short_name %q, got %q", "SimpleDMS", manifestData.ShortName)
	}
}

func TestPWAManifestBytesDoesNotOverrideWhenEnvIsEmpty(t *testing.T) {
	t.Setenv(devPWANameEnvVar, "   ")

	pwaManifestHandler := NewPWAManifestHandler(testManifestFS(), true)
	manifestBytes, err := pwaManifestHandler.manifestBytes()
	if err != nil {
		t.Fatalf("pwa manifest bytes: %v", err)
	}

	manifestData := parseManifestData(t, manifestBytes)
	if manifestData.Name != "SimpleDMS" {
		t.Fatalf("expected name %q, got %q", "SimpleDMS", manifestData.Name)
	}
	if manifestData.ShortName != "SimpleDMS" {
		t.Fatalf("expected short_name %q, got %q", "SimpleDMS", manifestData.ShortName)
	}
}

func TestPWAManifestBytesRemovesInternalCommentKey(t *testing.T) {
	pwaManifestHandler := NewPWAManifestHandler(testManifestFS(), false)
	manifestBytes, err := pwaManifestHandler.manifestBytes()
	if err != nil {
		t.Fatalf("pwa manifest bytes: %v", err)
	}

	manifestData := map[string]any{}
	err = json.Unmarshal(manifestBytes, &manifestData)
	if err != nil {
		t.Fatalf("unmarshal manifest map: %v", err)
	}

	if _, found := manifestData[pwaManifestInternalCommentKey]; found {
		t.Fatalf("expected %q to be removed from served manifest", pwaManifestInternalCommentKey)
	}
}

func TestPWAManifestBytesCachesResult(t *testing.T) {
	t.Setenv(devPWANameEnvVar, "First Name")

	pwaManifestHandler := NewPWAManifestHandler(testManifestFS(), true)
	manifestBytes, err := pwaManifestHandler.manifestBytes()
	if err != nil {
		t.Fatalf("pwa manifest bytes: %v", err)
	}

	manifestData := parseManifestData(t, manifestBytes)
	if manifestData.Name != "First Name" {
		t.Fatalf("expected name %q, got %q", "First Name", manifestData.Name)
	}

	t.Setenv(devPWANameEnvVar, "Second Name")
	manifestBytes, err = pwaManifestHandler.manifestBytes()
	if err != nil {
		t.Fatalf("pwa manifest bytes: %v", err)
	}

	manifestData = parseManifestData(t, manifestBytes)
	if manifestData.Name != "First Name" {
		t.Fatalf("expected cached name %q, got %q", "First Name", manifestData.Name)
	}
}

func testManifestFS() fstest.MapFS {
	return fstest.MapFS{
		"manifest.json": &fstest.MapFile{Data: []byte(`{"name":"SimpleDMS","short_name":"SimpleDMS","id":"/","x_simpledms_comment":"This file is processed by PWAManifestHandler before serving."}`)},
	}
}

type pwaManifestData struct {
	Name      string `json:"name"`
	ShortName string `json:"short_name"`
}

func parseManifestData(t *testing.T, manifestBytes []byte) pwaManifestData {
	t.Helper()

	manifestData := pwaManifestData{}
	err := json.Unmarshal(manifestBytes, &manifestData)
	if err != nil {
		t.Fatalf("unmarshal manifest: %v", err)
	}

	return manifestData
}
