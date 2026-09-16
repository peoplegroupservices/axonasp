package axonvm

import (
	"path/filepath"
	"strings"

	"github.com/peoplegroupservices/axonasp/v2/axonconfig"
)

// resolveConfiguredTempDir returns global.temp_dir with a safe project-local fallback.
func resolveConfiguredTempDir() string {
	tempDir := strings.TrimSpace(axonconfig.NewViper().GetString("global.temp_dir"))
	if tempDir == "" {
		tempDir = filepath.Join(".", "temp")
	}
	return filepath.Clean(tempDir)
}
