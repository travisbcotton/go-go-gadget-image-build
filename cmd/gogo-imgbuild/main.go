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
    //pkgs := []string{"pkg1","pkg2"}
    irepos := []string{"https://download.rockylinux.org/pub/rocky/8/BaseOS/x86_64/os/"}

    repos := []bootstrap.Repo{}
    for _,r := range irepos {
        repos = append(repos, bootstrap.Repo{
            BaseURL: strings.TrimSpace(r), 
            Arch: "x86_64",
        })
    }
    fmt.Println("Repo struct", repos)

    resolve := rpm.NewRepodataResolver(repos)
    m,err := resolve.Resolve(bootstrap.Spec{Raw: "bash"})
    if err != nil {
        panic(err)
    }
    fmt.Printf("Match:\n  Name: %s\n  EVR: %s\n  Arch: %s\n  URL:  %s\n  File: %s\n", m.Name, m.EVR, m.Arch, m.URL, m.File)
}