package main

import (
    "fmt"
    "os"
    "path/filepath"
    "strconv"

    storage "github.com/containers/storage"
)

func openStore() (storage.Store, error) {
    opts, err := storage.DefaultStoreOptions() // <- correct helper in v1.59.x
    if err != nil {
        return nil, fmt.Errorf("default store opts: %w", err)
    }

    // Tweak for rootless (optional but helpful)
    if os.Geteuid() != 0 {
        uid := os.Geteuid()
        runRoot := filepath.Join("/run/user", strconv.Itoa(uid))
        home, _ := os.UserHomeDir()
        graphRoot := filepath.Join(home, ".local/share/containers/storage")

        opts.RunRoot = runRoot
        opts.GraphRoot = graphRoot
        if opts.GraphDriverName == "" {
            opts.GraphDriverName = "overlay"
        }
    }

    return storage.GetStore(opts)
}
