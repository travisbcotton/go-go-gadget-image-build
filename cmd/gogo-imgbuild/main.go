package main

//"github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"

import (
    "fmt"
    "strings"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

func main() {
    //driver := rpm.NewRepodataResolver()
    ipkgs := []string{"bash","kernel"}
    irepos := []string{
        "https://download.rockylinux.org/pub/rocky/8/BaseOS/x86_64/os/",
        "https://download.rockylinux.org/pub/rocky/8/AppStream/x86_64/os/",
    }

    repos := []bootstrap.Repo{}
    for _,r := range irepos {
        repos = append(repos, bootstrap.Repo{
            BaseURL: strings.TrimSpace(r), 
            Arch: "x86_64",
        })
    }
    fmt.Println("Repo struct", repos)

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
}