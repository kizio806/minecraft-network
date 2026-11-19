package pluginmanager

import (
	"testing"
)

func TestValidateDetails(t *testing.T) {
	// Schema version checks
	m, l := validFixture()
	m.SchemaVersion = 2
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on manifest schema version != 1")
	}

	m, l = validFixture()
	l.SchemaVersion = 2
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on lockfile schema version != 1")
	}

	// Minecraft version
	m, l = validFixture()
	m.Environment.MinecraftVersion = "1.20.4"
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on unsupported minecraft version")
	}

	// Empty plugins
	m, l = validFixture()
	m.Plugins = map[string]PluginSpec{}
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on empty plugins")
	}

	// Invalid ID
	m, l = validFixture()
	m.Plugins["INVALID ID!"] = m.Plugins["example"]
	delete(m.Plugins, "example")
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on invalid plugin ID")
	}

	// Unsupported provider
	m, l = validFixture()
	p := m.Plugins["example"]
	p.Source = "unknown_provider"
	m.Plugins["example"] = p
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on unsupported provider")
	}

	// Update policy must be manual
	m, l = validFixture()
	p = m.Plugins["example"]
	p.UpdatePolicy = "auto"
	m.Plugins["example"] = p
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on non-manual update policy")
	}

	// Unsupported platform
	m, l = validFixture()
	p = m.Plugins["example"]
	p.Platform = "spigot"
	m.Plugins["example"] = p
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on unsupported platform")
	}

	// Unsupported server target
	m, l = validFixture()
	p = m.Plugins["example"]
	p.Servers = []string{"invalid_server"}
	m.Plugins["example"] = p
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on unsupported server target")
	}

	// Missing from lockfile
	m, l = validFixture()
	delete(l.Plugins, "example")
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on missing lockfile entry")
	}

	// Lockfile mismatch: Project
	m, l = validFixture()
	e := l.Plugins["example"]
	e.Project = "wrong_proj"
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on lockfile project mismatch")
	}

	// Lockfile missing digest
	m, l = validFixture()
	e = l.Plugins["example"]
	e.SHA256 = ""
	e.SHA512 = ""
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on missing digests")
	}

	// Lockfile malformed sha256
	m, l = validFixture()
	e = l.Plugins["example"]
	e.SHA256 = "not_valid_hex_digest"
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on malformed sha256")
	}

	// Game version mismatch
	m, l = validFixture()
	e = l.Plugins["example"]
	e.GameVersions = []string{"1.20.1"}
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on game version mismatch")
	}

	// Platform loader mismatch
	m, l = validFixture()
	e = l.Plugins["example"]
	e.Loaders = []string{"paper"}
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on loader platform mismatch")
	}

	// Dependencies mismatch
	m, l = validFixture()
	e = l.Plugins["example"]
	e.Dependencies = []string{"unexpected_dep"}
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on dependencies mismatch")
	}

	// Unsafe download URL (non-https)
	m, l = validFixture()
	e = l.Plugins["example"]
	e.DownloadURL = "http://cdn.modrinth.com/example.jar"
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on HTTP download URL")
	}

	// Floating download URL
	m, l = validFixture()
	e = l.Plugins["example"]
	e.DownloadURL = "https://cdn.modrinth.com/example/latest/download"
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on floating URL")
	}

	// Unsafe filename (path traversal)
	m, l = validFixture()
	e = l.Plugins["example"]
	e.Filename = "../example.jar"
	l.Plugins["example"] = e
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on unsafe filename")
	}

	// Undeclared plugin in lockfile
	m, l = validFixture()
	l.Plugins["extra_plugin"] = l.Plugins["example"]
	if err := Validate(m, l); err == nil {
		t.Fatal("expected error on undeclared lockfile entry")
	}
}

func TestValidatorHelpers(t *testing.T) {
	// validID
	if !validID("test_plugin-1") || validID("") || validID("UPPERCASE") || validID("test!@#") {
		t.Fatal("validID check failed")
	}

	// hexDigest
	if !hexDigest("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 64) {
		t.Fatal("expected valid 64-char hex digest")
	}
	if hexDigest("not-hex", 64) || hexDigest("aa", 64) {
		t.Fatal("expected invalid hex digest")
	}

	// contains
	if !contains([]string{"a", "b", "c"}, "b") || contains([]string{"a", "b"}, "z") {
		t.Fatal("contains check failed")
	}

	// sameStrings
	if !sameStrings([]string{"a", "b"}, []string{"a", "b"}) {
		t.Fatal("expected sameStrings to match")
	}
	if sameStrings([]string{"a"}, []string{"a", "b"}) || sameStrings([]string{"a", "b"}, []string{"a", "c"}) {
		t.Fatal("expected sameStrings to differ")
	}

	// floatingURL
	if !floatingURL("https://host/releases/latest") || !floatingURL("https://host/v?version=latest") || floatingURL("https://host/v1.0.0.jar") {
		t.Fatal("floatingURL check failed")
	}
}

func TestDependencyOrder(t *testing.T) {
	// Valid chain: a -> b -> c
	valid := map[string]PluginSpec{
		"a": {Dependencies: []string{"b"}},
		"b": {Dependencies: []string{"c"}},
		"c": {Dependencies: nil},
	}
	if err := dependencyOrder(valid); err != nil {
		t.Fatalf("dependencyOrder failed on valid chain: %v", err)
	}

	// Diamond: a -> (b, c) -> d
	diamond := map[string]PluginSpec{
		"a": {Dependencies: []string{"b", "c"}},
		"b": {Dependencies: []string{"d"}},
		"c": {Dependencies: []string{"d"}},
		"d": {Dependencies: nil},
	}
	if err := dependencyOrder(diamond); err != nil {
		t.Fatalf("dependencyOrder failed on diamond graph: %v", err)
	}

	// Missing dependency
	missing := map[string]PluginSpec{
		"a": {Dependencies: []string{"unknown"}},
	}
	if err := dependencyOrder(missing); err == nil {
		t.Fatal("expected error on missing dependency")
	}

	// Direct cycle: a -> b -> a
	cycle := map[string]PluginSpec{
		"a": {Dependencies: []string{"b"}},
		"b": {Dependencies: []string{"a"}},
	}
	if err := dependencyOrder(cycle); err == nil {
		t.Fatal("expected error on dependency cycle")
	}
}
