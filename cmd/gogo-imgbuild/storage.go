// storage.go
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	storage "github.com/containers/storage"
)

func openStore() (storage.Store, error) {
	opts, err := storage.DefaultStoreOptionsAutoDetect()
	if err != nil {
		return nil, fmt.Errorf("default store opts: %w", err)
	}

	// If running rootless, point to XDG dirs
	if os.Geteuid() != 0 {
		uid := os.Geteuid()
		runRoot := filepath.Join("/run/user", strconv.Itoa(uid))
		home, _ := os.UserHomeDir()
		graphRoot := filepath.Join(home, ".local/share/containers/storage")

		opts.RunRoot = runRoot
		opts.GraphRoot = graphRoot
		// Overlay is best; fallback to vfs if overlay unavailable.
		if opts.GraphDriverName == "" {
			opts.GraphDriverName = "overlay"
		}
	}

	// Respect $CONTAINERS_STORAGE_CONF if present (Buildah/Podman standard)
	if conf := os.Getenv("CONTAINERS_STORAGE_CONF"); conf != "" {
		opts.GraphDriverOptions = append(opts.GraphDriverOptions, fmt.Sprintf("mount_program=%s", conf))
	}

	return storage.GetStore(opts)
}
