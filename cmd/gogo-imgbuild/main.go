package main
/*

*/
import (
    "fmt"
    "net/http"
    "time"
    "context"
    "log"
    "flag"

	storageRef "github.com/containers/image/v5/storage"
	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage/pkg/reexec"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/internal/config"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

func main() {
    if reexec.Init() { return }
    defaultCfg := "./bootstrap.yaml"
    cfgPath := flag.String("config", defaultCfg, "path to YAML config (or '-' for stdin)")
    flag.Parse()

    
    bctx := context.Background()
    store, err := openStore()
    if err != nil { log.Fatal(err) }
	defer store.Shutdown(false)
    builder, err := buildah.NewBuilder(bctx, store, buildah.BuilderOptions{
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
            ID: r.ID,
            GPG: r.GPG,
            GPGCheck: *r.GPGCheck,
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

    //Create /etc/os-release
    err = rpm.WriteOSRelease(rootfs, rpm.OSRelease{
        Name:       "Distroless",
        ID:         "distroless",
        VersionID:  "9",
        PrettyName: "Distroless Minimal",
    })
    if err != nil {
        panic(err)
    }

    //Write repos to /etc/yum.repos.d/gogo-imgbuild.repo
    err = rpm.WriteRepos(rootfs, repos)
    if err != nil {
        panic(err)
    }

    fmt.Println("Unmounting Container")
    if err := builder.Unmount(); err != nil {
		log.Printf("unmount warning: %v", err)
	}

    //Run commands in container
    if len(cfg.Cmds) > 0 {
        fmt.Println("Running commands")
        for _, c := range(cfg.Cmds) {
            _ = runInContainer(builder, c)
        }
    }

    imageName := "localhost/custom-base:latest"
    dest, err := storageRef.Transport.ParseStoreReference(store, imageName)
    if err != nil {
	panic(err)
    }
    fmt.Println("Committing Container")
    _, _, _, err = builder.Commit(bctx, dest, buildah.CommitOptions{
		PreferredManifestType: define.OCIv1ImageManifest,
	})
    if err != nil { log.Fatalf("commit: %v", err) }
    fmt.Println("Committed image:", imageName)
}