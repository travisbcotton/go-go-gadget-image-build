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
            "TERM=xterm-256color",
        },
		AddCapabilities: []string{
        "CAP_CHOWN","CAP_DAC_OVERRIDE","CAP_FOWNER","CAP_FSETID","CAP_KILL",
        "CAP_NET_BIND_SERVICE","CAP_SETFCAP","CAP_SETGID","CAP_SETPCAP","CAP_SETUID","CAP_SYS_CHROOT",
        },
    }

    argv := []string{"/bin/sh", "-lc", script}
    err := b.Run(argv, opts)
    return errb.String(), err
}