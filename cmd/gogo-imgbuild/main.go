package main

import (
    "fmt"
    "strings"
    "net/http"
    "time"
    "context"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

func main() {
    ipkgs := []string{
        "bash",
        "libdnf",
    }
    irepos := []string{
        "https://download.rockylinux.org/pub/rocky/9/BaseOS/x86_64/os/",
        "https://download.rockylinux.org/pub/rocky/9/AppStream/x86_64/os/",
        "https://dl.rockylinux.org/pub/rocky/9/CRB/x86_64/os",
    }

    repos := []bootstrap.Repo{}
    for _,r := range irepos {
        repos = append(repos, bootstrap.Repo{
            BaseURL: strings.TrimSpace(r), 
            Arch: "x86_64",
        })
    }

    pkgs := bootstrap.Spec{}
    for _, p := range ipkgs {
        pkgs.Raw = append(pkgs.Raw, p)
    }

    resolve := rpm.NewRepodataResolver(repos)
    matches,err := resolve.Resolve(pkgs)
    if err != nil {
        panic(err)
    }

    for _, m := range matches {
        if m.Name != "" {
            fmt.Printf("Match:\n  Name: %s\n  EVR: %s\n  Arch: %s\n  URL:  %s\n  File: %s\n", m.Name, m.EVR, m.Arch, m.URL, m.File)
        }
    }

    var rpms []string
    getter := rpm.NewGetterDownloader(&http.Client{Timeout: 45 * time.Second})
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    for _, m := range matches {
        res, err := getter.DownloadRPM(ctx, m.URL, "./rpms")
        if err != nil {
            fmt.Println("failed to download RPM")
            panic(err)
        }
        fmt.Println("filepath:", res.Path)
        rpms = append(rpms, res.Path)
    }

    err = rpm.InstallRPMs(rpms,"./rootfs")
    if err != nil {
        panic(err)
    }
}