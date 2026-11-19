package pluginmanager

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type change struct {
	destination, backup string
	existed             bool
}

func rollback(changes []change) error {
	var errs []error
	for i := len(changes) - 1; i >= 0; i-- {
		c := changes[i]
		if c.existed {
			if err := os.Rename(c.backup, c.destination); err != nil {
				errs = append(errs, err)
			}
		} else {
			if err := os.Remove(c.destination); err != nil && !os.IsNotExist(err) {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

func (m *Manager) writeRollback(changes []change) error {
	files := make([]RollbackFile, 0, len(changes))
	for _, c := range changes {
		files = append(files, RollbackFile{Destination: c.destination, Backup: c.backup, Existed: c.existed})
	}
	b, err := yaml.Marshal(RollbackState{CreatedAt: time.Now().UTC().Format(time.RFC3339), Files: files})
	if err != nil {
		return err
	}
	tmp := filepath.Join(m.root, ".plugin-manager", ".rollback.yml.tmp")
	if err := os.WriteFile(tmp, b, 0600); err != nil {
		return err
	}
	return os.Rename(tmp, filepath.Join(m.root, ".plugin-manager", "rollback.yml"))
}

func (m *Manager) Rollback() error {
	state, err := decode[RollbackState](filepath.Join(m.root, ".plugin-manager", "rollback.yml"))
	if err != nil {
		return fmt.Errorf("no rollback snapshot: %w", err)
	}
	if err := m.ensureStopped(context.Background(), []string{"proxy", "lobby", "survival"}); err != nil {
		return err
	}
	for i := len(state.Files) - 1; i >= 0; i-- {
		f := state.Files[i]
		if f.Existed {
			if err := os.Rename(f.Backup, f.Destination); err != nil {
				return err
			}
		} else if err := os.Remove(f.Destination); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
