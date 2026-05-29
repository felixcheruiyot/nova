// Package pkgmgr manages nova.mod and the nova_modules directory.
package pkgmgr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const modFile = "nova.mod"
const modulesDir = "nova_modules"

// Mod is the structure of nova.mod.
type Mod struct {
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	Dependencies map[string]string `json:"dependencies"`
}

func readMod() (*Mod, error) {
	data, err := os.ReadFile(modFile)
	if os.IsNotExist(err) {
		cwd, _ := os.Getwd()
		return &Mod{
			Name:         filepath.Base(cwd),
			Version:      "0.1.0",
			Dependencies: map[string]string{},
		}, nil
	}
	if err != nil {
		return nil, err
	}
	var m Mod
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if m.Dependencies == nil {
		m.Dependencies = map[string]string{}
	}
	return &m, nil
}

func writeMod(m *Mod) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(modFile, append(data, '\n'), 0644)
}

// Add records pkg in nova.mod and creates a stub in nova_modules/ if needed.
func Add(pkg string) error {
	m, err := readMod()
	if err != nil {
		return err
	}
	if _, exists := m.Dependencies[pkg]; exists {
		fmt.Printf("Package %q is already in nova.mod\n", pkg)
		return nil
	}

	m.Dependencies[pkg] = "latest"
	if err := writeMod(m); err != nil {
		return err
	}

	// Ensure nova_modules directory exists
	if err := os.MkdirAll(modulesDir, 0755); err != nil {
		return err
	}

	// Create a stub module file if not already present
	stubPath := filepath.Join(modulesDir, pkg+".nv")
	if _, err := os.Stat(stubPath); os.IsNotExist(err) {
		stub := fmt.Sprintf("# Package: %s\n# Place your implementation here or replace with a real package.\n", pkg)
		if err := os.WriteFile(stubPath, []byte(stub), 0644); err != nil {
			return err
		}
		fmt.Printf("Created stub: %s\n", stubPath)
	}

	fmt.Printf("Added %q to nova.mod\n", pkg)
	fmt.Println("Note: the Nova package registry is not live yet. Add your implementation to", stubPath)
	return nil
}

// Remove removes pkg from nova.mod.
func Remove(pkg string) error {
	m, err := readMod()
	if err != nil {
		return err
	}
	if _, exists := m.Dependencies[pkg]; !exists {
		return fmt.Errorf("package %q is not in nova.mod", pkg)
	}
	delete(m.Dependencies, pkg)
	if err := writeMod(m); err != nil {
		return err
	}
	fmt.Printf("Removed %q from nova.mod\n", pkg)

	// Optionally remove the stub (only if it's a stub we created)
	stubPath := filepath.Join(modulesDir, pkg+".nv")
	if data, err := os.ReadFile(stubPath); err == nil {
		if len(data) == 0 || string(data[:2]) == "# " {
			_ = os.Remove(stubPath)
			fmt.Printf("Removed %s\n", stubPath)
		}
	}
	return nil
}

// List prints all declared dependencies.
func List() error {
	m, err := readMod()
	if err != nil {
		return err
	}
	if len(m.Dependencies) == 0 {
		fmt.Println("No dependencies in nova.mod")
		return nil
	}
	fmt.Printf("Dependencies in %s (%s %s):\n", modFile, m.Name, m.Version)
	for pkg, ver := range m.Dependencies {
		fmt.Printf("  %s  %s\n", pkg, ver)
	}
	return nil
}
