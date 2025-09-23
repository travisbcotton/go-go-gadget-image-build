package main

import (
	"os"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
)

func runInContainer(b *buildah.Builder, argv []string) error {
	opts := buildah.RunOptions{
		// chroot isolation keeps it simple (no runc/netavark needed)
		Isolation: define.IsolationChroot,

		// Wire stdio so you see output / can interact
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Stdin:  os.Stdin,

	}
	return b.Run(argv, opts)
}