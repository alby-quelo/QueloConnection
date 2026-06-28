package configfile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func LoadSavedMachines(_ string) []string {
	return ListSavedMachines()
}

func ListSavedMachines() []string {
	data, err := os.ReadFile(SavedMachinesPath())
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, line := range splitLines(string(data)) {
		name := strings.TrimSpace(line)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		out = append(out, name)
	}
	return out
}

func HasSavedMachine(name string) bool {
	name = strings.TrimSpace(name)
	for _, m := range ListSavedMachines() {
		if m == name {
			return true
		}
	}
	return false
}

func DeleteSavedMachine(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return os.ErrInvalid
	}
	remaining := ListSavedMachines()
	var kept []string
	found := false
	for _, m := range remaining {
		if m == name {
			found = true
			continue
		}
		kept = append(kept, m)
	}
	if !found {
		return fmt.Errorf("macchina non presente nell'elenco")
	}
	path := SavedMachinesPath()
	if len(kept) == 0 {
		return os.Remove(path)
	}
	var b strings.Builder
	for _, m := range kept {
		b.WriteString(m)
		b.WriteByte('\n')
	}
	return os.WriteFile(path, []byte(b.String()), 0o600)
}

func SaveMachine(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return os.ErrInvalid
	}
	if HasSavedMachine(name) {
		return nil
	}
	dir := filepath.Dir(SavedMachinesPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	f, err := os.OpenFile(SavedMachinesPath(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(name + "\n")
	return err
}
