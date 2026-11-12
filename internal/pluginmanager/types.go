package pluginmanager

import "time"

type Manifest struct {
	SchemaVersion int                   `yaml:"schema-version"`
	Environment   Environment           `yaml:"environment"`
	Plugins       map[string]PluginSpec `yaml:"plugins"`
}

type Environment struct {
	MinecraftVersion string `yaml:"minecraft-version"`
	Backend          string `yaml:"backend"`
	Proxy            string `yaml:"proxy"`
	Java             string `yaml:"java"`
}

type PluginSpec struct {
	DisplayName      string   `yaml:"display-name"`
	Enabled          bool     `yaml:"enabled"`
	Source           string   `yaml:"source"`
	Project          string   `yaml:"project"`
	Version          string   `yaml:"version"`
	VersionID        string   `yaml:"version-id"`
	Platform         string   `yaml:"platform"`
	Servers          []string `yaml:"servers"`
	ReleaseChannel   string   `yaml:"release-channel"`
	UpdatePolicy     string   `yaml:"update-policy"`
	Dependencies     []string `yaml:"dependencies"`
	IncompatibleWith []string `yaml:"incompatible-with"`
	RequiredJava     string   `yaml:"required-java"`
	ConfigDir        string   `yaml:"config-dir"`
	ConfigTargetDir  string   `yaml:"config-target-dir"`
	License          string   `yaml:"license"`
}

type Lockfile struct {
	SchemaVersion int                  `yaml:"schema-version"`
	GeneratedBy   string               `yaml:"generated-by"`
	Plugins       map[string]LockEntry `yaml:"plugins"`
}

type ComponentLockfile struct {
	SchemaVersion int                           `yaml:"schema-version"`
	Components    map[string]ComponentLockEntry `yaml:"components"`
}

type ComponentLockEntry struct {
	Version     string   `yaml:"version"`
	Tag         string   `yaml:"tag"`
	Source      string   `yaml:"source"`
	Repository  string   `yaml:"repository"`
	DownloadURL string   `yaml:"download-url"`
	Filename    string   `yaml:"filename"`
	SHA256      string   `yaml:"sha256"`
	Minecraft   []string `yaml:"minecraft"`
}

type LockEntry struct {
	Project      string   `yaml:"project"`
	Version      string   `yaml:"version"`
	VersionID    string   `yaml:"version-id"`
	Source       string   `yaml:"source"`
	DownloadURL  string   `yaml:"download-url"`
	Filename     string   `yaml:"filename"`
	SHA256       string   `yaml:"sha256"`
	SHA512       string   `yaml:"sha512"`
	Size         int64    `yaml:"size"`
	GameVersions []string `yaml:"game-versions"`
	Loaders      []string `yaml:"loaders"`
	Dependencies []string `yaml:"dependencies"`
	ResolvedAt   string   `yaml:"resolved-at"`
}

type InstalledState struct {
	InstalledAt time.Time            `yaml:"installed-at"`
	Plugins     map[string]LockEntry `yaml:"plugins"`
}

type Artifact struct {
	Entry LockEntry
	Path  string
}

type RollbackState struct {
	CreatedAt string         `yaml:"created-at"`
	Files     []RollbackFile `yaml:"files"`
}

type RollbackFile struct {
	Destination string `yaml:"destination"`
	Backup      string `yaml:"backup"`
	Existed     bool   `yaml:"existed"`
}
