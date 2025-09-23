package main

import (
	"os"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
)

func runInContainer(b *buildah.Builder, script string) (string, string, error) {
    var out, errb bytes.Buffer

    opts := buildah.RunOptions{
        Isolation:     define.IsolationChroot,
        NetworkPolicy: define.NetworkDefault,
        Stdout:        &out,
        Stderr:        &errb,
        Env: map[string]string{
            "PATH":      "/usr/sbin:/usr/bin:/sbin:/bin",
            "HOME":      "/root",
            "TMPDIR":    "/var/tmp",
            "container": "oci",
            "TERM":      "xterm-256color",
        },
    }

    argv := []string{"/bin/sh", "-lc", script}
    err := b.Run(argv, opts)
    return out.String(), errb.String(), err
}