package compose

import (
	"context"
	"io"
)

// ManagedServices defines core backend services expected to run and be healthy.
var ManagedServices = []string{"proxy", "auth", "lobby", "survival"}

// Client coordinates operations against Docker Compose.
type Client struct {
	Runner      Runner
	ProjectDir  string
	ComposeFile string
}

// NewClient initializes a Compose client.
func NewClient(runner Runner, projectDir, composeFile string) *Client {
	if runner == nil {
		runner = ExecRunner{}
	}
	return &Client{
		Runner:      runner,
		ProjectDir:  projectDir,
		ComposeFile: composeFile,
	}
}

// BuildArgs constructs the base docker compose command line slice.
func (c *Client) BuildArgs(profile string, args ...string) []string {
	var base []string
	if profile != "" {
		base = []string{"compose", "-f", c.ComposeFile, "--profile", profile}
	} else {
		base = []string{"compose", "-f", c.ComposeFile}
	}
	return append(base, args...)
}

// Compose executes a docker compose command and returns stdout/stderr.
func (c *Client) Compose(ctx context.Context, profile string, args ...string) (string, error) {
	cmdArgs := c.BuildArgs(profile, args...)
	return c.Runner.Run(ctx, c.ProjectDir, "docker", cmdArgs...)
}

// ComposeStream executes a docker compose command streaming stdout and stderr.
func (c *Client) ComposeStream(ctx context.Context, out, errOut io.Writer, profile string, args ...string) error {
	cmdArgs := c.BuildArgs(profile, args...)
	return c.Runner.RunStream(ctx, c.ProjectDir, out, errOut, "docker", cmdArgs...)
}

// IsMonitoringService checks whether a service belongs to the monitoring stack.
func IsMonitoringService(s string) bool {
	switch s {
	case "node-exporter", "prometheus", "grafana", "uptime-kuma", "minecraft-exporter":
		return true
	}
	return false
}

// ValidService verifies if a service identifier is supported by the network.
func ValidService(s string) bool {
	switch s {
	case "proxy", "auth", "lobby", "survival", "postgres", "node-exporter", "prometheus", "grafana", "uptime-kuma", "minecraft-exporter":
		return true
	}
	return false
}

// IsConsoleService checks if interactive console attachment is supported.
func IsConsoleService(s string) bool {
	switch s {
	case "proxy", "lobby", "survival":
		return true
	}
	return false
}
