package compose

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ComposeRow holds inspection fields returned by docker compose ps --format json.
type ComposeRow struct {
	Service string `json:"Service"`
	State   string `json:"State"`
	Health  string `json:"Health"`
	Status  string `json:"Status"`
	Name    string `json:"Name"`
}

// ParseComposeRows unmarshals line-delimited JSON objects into ComposeRow slices.
func ParseComposeRows(out string) ([]ComposeRow, error) {
	rows := []ComposeRow{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		var row ComposeRow
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, fmt.Errorf("invalid Compose status JSON: %w", err)
		}
		rows = append(rows, row)
	}
	return rows, nil
}

// HealthStatus aggregates running status and health evaluations for services.
type HealthStatus struct {
	Services []ComposeRow
	AllOK    bool
}

// CheckHealth queries running services and assesses readiness of managed services.
func (c *Client) CheckHealth(ctx context.Context) (*HealthStatus, error) {
	out, err := c.Compose(ctx, "", "ps", "--format", "json")
	if err != nil {
		return nil, err
	}
	rows, err := ParseComposeRows(out)
	if err != nil {
		return nil, err
	}
	state := map[string]ComposeRow{}
	for _, row := range rows {
		state[row.Service] = row
	}
	allOK := true
	for _, service := range ManagedServices {
		row, ok := state[service]
		if !ok || row.State != "running" || (row.Health != "" && row.Health != "healthy") {
			allOK = false
		}
	}
	return &HealthStatus{Services: rows, AllOK: allOK}, nil
}
