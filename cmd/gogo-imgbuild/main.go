package main

import (
    "fmt"
    "net/http"
    "time"
    "context"
    "log"
    "flag"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/internal/config"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

func main() {
    defaultCfg := "./bootstrap.yaml"
    cfgPath := flag.String("config", defaultCfg, "path to YAML config (or '-' for stdin)")
    flag.Parse()

    // load config file
    cfg, err := config.Load(*cfgPath)
    if err != nil { log.Fatal(err) }

    //Populate repos
    repos := make([]bootstrap.Repo, 0, len(cfg.Repos))
    for _,r := range cfg.Repos {
        repos = append(repos, bootstrap.Repo{
            BaseURL: r.URL, 
        })
    }

    //Populate packages
    pkgs := bootstrap.Package{}
    for _, p := range cfg.Packages {
        pkgs.Raw = append(pkgs.Raw, p)
    }

    //Set arch
    arch := cfg.Arch

    //Find best matches
    resolve := rpm.NewRepodataResolver(repos, arch)
    matches,err := resolve.Resolve(pkgs)
    if err != nil {
        panic(err)
    }

    //Print matches
    for _, m := range matches {
        if m.Name != "" {
            fmt.Printf("Match:\n  Name: %s\n  EVR: %s\n  Arch: %s\n  URL:  %s\n  File: %s\n", m.Name, m.EVR, m.Arch, m.URL, m.File)
        }
    }

    //Download all best matches
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

    //Install packages
    err = rpm.InstallRPMs(rpms,"./rootfs")
    if err != nil {
        panic(err)
    }
}