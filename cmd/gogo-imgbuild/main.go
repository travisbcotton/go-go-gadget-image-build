package main

import (
    "fmt"
    "net/http"
    "time"
    "context"
    "log"
    "flag"

	storage "github.com/containers/storage"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/internal/config"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

func main() {
    defaultCfg := "./bootstrap.yaml"
    cfgPath := flag.String("config", defaultCfg, "path to YAML config (or '-' for stdin)")
    flag.Parse()

    store, err := openStore()
    if err != nil { log.Fatal(err) }
	defer store.Shutdown(false)
    builder, err := buildah.NewBuilder(ctx, store, buildah.BuilderOptions{
		FromImage: "scratch",
	})
    if err != nil { log.Fatalf("new builder: %v", err) }
    defer func() { _ = builder.Delete() }()
    mountPoint, err := builder.Mount("")
    if err != nil { log.Fatalf("mount: %v", err) }
    fmt.Println("Mounted at:", mountPoint)
    rootfs := mountPoint
    //_ = runCommandInChroot(rootfs, "rpm", "--initdb")

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
    fmt.Println("Found matches")
    for _, m := range matches {
       if m.Name != "" {
            fmt.Printf("Package: %s\n", m.File)
        }
    }

    //Download all best matches
    var rpms []string
    getter := rpm.NewGetterDownloader(&http.Client{Timeout: 45 * time.Second})
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    for _, m := range matches {
        fmt.Println("Downloading:", m.Name)
        res, err := getter.DownloadRPM(ctx, m.URL, "./rpms")
        if err != nil {
            fmt.Println("failed to download RPM")
            panic(err)
        }
        rpms = append(rpms, res.Path)
    }

    //Install packages
    err = rpm.InstallRPMs(rpms, rootfs)
    if err != nil {
        panic(err)
    }

    if _, err := builder.Unmount(); err != nil {
		log.Printf("unmount warning: %v", err)
	}

    imageName := "localhost/custom-base:latest"
    _, _, err = builder.Commit(ctx, imageName, buildah.CommitOptions{
		PreferredManifestType: define.OCIv1ImageManifest,
	})
    if err != nil { log.Fatalf("commit: %v", err) }
    fmt.Println("Committed image:", imageName)
}