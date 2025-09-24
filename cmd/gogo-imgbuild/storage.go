package main

import (
    "fmt"

    storage "github.com/containers/storage"
)

func openStore() (storage.Store, error) {
    opts, err := storage.DefaultStoreOptions()
    if err != nil {
        return nil, fmt.Errorf("default store opts: %w", err)
    }

    opts.GraphRoot = "/home/builder/.local/share/containers/storage"
    opts.RunRoot = "/var/tmp/storage-run-1000/containers"
    opts.GraphDriverName = "overlay"
	opts.RootlessStoragePath = ""

    return storage.GetStore(opts)
}
