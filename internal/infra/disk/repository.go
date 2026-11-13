package disk

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

func readYamlFile(path string, data any) error {
	if err := checkFile(path); err != nil {
		return fmt.Errorf("failed to check %s file", path)
	}
	fileData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}
	if err := yaml.Unmarshal(fileData, data); err != nil {
		return err
	}
	return nil
}

func readJsonFile(path string, data any) error {
	if err := checkFile(path); err != nil {
		return fmt.Errorf("failed to check %s file", path)
	}
	fileData, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}
	if err := json.Unmarshal(fileData, data); err != nil {
		return err
	}
	return nil
}

func checkFile(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := writeFileAtomically(path, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func writeYamlFile(path string, data any) error {
	dataFile, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return writeFileAtomically(path, dataFile)
}

func writeJsonFile(path string, data any) error {
	dataFile, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}
	return writeFileAtomically(path, dataFile)
}

const (
	filePermissions = 0o644
	DirPermissions  = 0o750
	tempFileSuffix  = ".tmp"
	tempFilePrefix  = "."
)

func writeFileAtomically(path string, data []byte) (err error) {
	dir, name := filepath.Split(path)
	if dir == "" {
		dir = "."
	}
	tmp, err := os.CreateTemp(dir, tempFilePrefix+name+".*"+tempFileSuffix)
	if err != nil {
		return fmt.Errorf("failed to create temporary file for %s: %w", path, err)
	}
	defer func() {
		if err != nil {
			_ = tmp.Close()
			_ = os.Remove(tmp.Name())
		}
	}()
	if _, err = tmp.Write(data); err != nil {
		return fmt.Errorf("failed to write temporary file for %s: %w", path, err)
	}
	if err = tmp.Chmod(filePermissions); err != nil {
		return fmt.Errorf("failed to set permissions on temporary file for %s: %w", path, err)
	}
	if err = tmp.Sync(); err != nil {
		return fmt.Errorf("failed to sync temporary file for %s: %w", path, err)
	}
	if err = tmp.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file for %s: %w", path, err)
	}
	if err = os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("failed to replace %s: %w", path, err)
	}
	return syncDir(dir)
}

func syncDir(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return fmt.Errorf("failed to open directory %s: %w", dir, err)
	}
	defer func() {
		_ = d.Close()
	}()
	if err = d.Sync(); err != nil {
		return fmt.Errorf("failed to sync directory %s: %w", dir, err)
	}
	return nil
}
