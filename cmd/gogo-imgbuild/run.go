package main

import (
	"bytes"
	"os"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
)

func runInContainer(b *buildah.Builder, script string) (string, error) {
    var errb bytes.Buffer

    opts := buildah.RunOptions{
        Isolation:     define.IsolationChroot,
		Stdin:		   os.Stdin,
        Stdout:        os.Stdout,
        Stderr:        &errb,
        Env: []string{
            "PATH=/usr/sbin:/usr/bin:/sbin:/bin",
            "HOME=/root",
            "TMPDIR=/var/tmp",
            "container=oci",
            "TERM=xterm-256color",
        },
    }

    argv := []string{"/bin/sh", "-lc", script}
    err := b.Run(argv, opts)
    return errb.String(), err
}