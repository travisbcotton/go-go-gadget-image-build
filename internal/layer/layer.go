package layer

import (
	"bytes"
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	"github.com/containers/storage"

	"github.com/travisbcotton/go-go-gadget-image-build/internal/config"

	"github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
	"github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

type Layer struct {
	Name    string
	Builder *buildah.Builder
	Store   storage.Store
}

func NewLayer(name string, parent string) (*Layer, error) {
	ctx := context.Background()
	store, err := openStore()
	if err != nil {
		log.Fatal(err)
	}
	builder, err := buildah.NewBuilder(ctx, store, buildah.BuilderOptions{
		FromImage: parent,
	})
	if err != nil {
		log.Fatalf("new builder: %v", err)
		return nil, err
	}
	return &Layer{
		Name:    name,
		Builder: builder,
		Store:   store,
	}, nil
}

func (b *Layer) BuildLayer(config config.Config) error {
	//Process repos from config file
	repos := make([]bootstrap.Repo, 0, len(config.Repos))
	for _, r := range config.Repos {
		repos = append(repos, bootstrap.Repo{
			BaseURL:  r.URL,
			ID:       r.ID,
			GPG:      r.GPG,
			GPGCheck: *r.GPGCheck,
		})
	}

	//Process packages
	pkgs := bootstrap.Package{}
	for _, p := range config.Packages {
		pkgs.Raw = append(pkgs.Raw, p)
	}

	//Set arch
	arch := config.Arch

	//Mount Container
	rootfs, err := b.Builder.Mount("")
	if err != nil {
		log.Fatalf("mount: %v", err)
		return err
	}
	log.Printf("Mounted at:", rootfs)

	// change behavior if importing from scratch
	if config.Opts.Parent == "scratch" {

		err = installIntoScratch(repos, pkgs, rootfs, arch)
		if err != nil {
			log.Fatalf("Failed to create layer from scratch %v", err)
		}

	} else {
		install_cmd, err := installIntoExisting(repos, pkgs, rootfs)
		if err != nil {
			log.Fatalf("Failed to create layer from parent %v", err)
		}
		b.RunInContainer(install_cmd)
	}

	log.Println("Unmounting Container")
	if err := b.Builder.Unmount(); err != nil {
		log.Printf("unmount warning: %v", err)
	}

	//Run commands in container
	if len(config.Cmds) > 0 {
		log.Println("Running commands")
		for _, c := range config.Cmds {
			errOut, err := b.RunInContainer(c)
			log.Println(errOut)
			if err != nil {
				log.Println(errOut)
			}
		}
	}
	return nil
}

func (b *Layer) RunInContainer(script string) (string, error) {
	var errb bytes.Buffer

	opts := buildah.RunOptions{
		Isolation: define.IsolationChroot,
		Stdin:     os.Stdin,
		Stdout:    os.Stdout,
		Stderr:    &errb,
		Env: []string{
			"PATH=/usr/sbin:/usr/bin:/sbin:/bin",
			"HOME=/root",
			"TMPDIR=/var/tmp",
			"TERM=xterm-256color",
		},
		AddCapabilities: []string{
			"CAP_CHOWN", "CAP_DAC_OVERRIDE", "CAP_FOWNER", "CAP_FSETID", "CAP_KILL",
			"CAP_NET_BIND_SERVICE", "CAP_SETFCAP", "CAP_SETGID", "CAP_SETPCAP", "CAP_SETUID", "CAP_SYS_CHROOT",
		},
	}

	argv := []string{"/bin/sh", "-lc", script}
	err := b.Builder.Run(argv, opts)
	return errb.String(), err
}

func installIntoScratch(repos []bootstrap.Repo, packages bootstrap.Package, rootfs string, arch []string) error {
	//create new resolver
	resolve := rpm.NewRepodataResolver(repos, arch)
	//find matches
	matches, err := resolve.Resolve(packages)
	if err != nil {
		return err
	}

	var rpms []string

	//Download packages
	getter := rpm.NewGetterDownloader(&http.Client{Timeout: 45 * time.Second})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	for _, m := range matches {
		log.Println("Downloading:", m.Name)
		res, err := getter.DownloadRPM(ctx, m.URL, "./rpms")
		if err != nil {
			log.Println("failed to download RPM")
			return err
		}
		rpms = append(rpms, res.Path)
	}

	//Install packages
	err = rpm.InstallRPMs(rpms, rootfs)
	if err != nil {
		return err
	}

	//Create /etc/os-release
	err = rpm.WriteOSRelease(rootfs, rpm.OSRelease{
		Name:       "Distroless",
		ID:         "distroless",
		VersionID:  "9",
		PrettyName: "Distroless Minimal",
	})
	if err != nil {
		return err
	}

	//Write repos to /etc/yum.repos.d/gogo-imgbuild.repo
	err = rpm.WriteRepos(rootfs, repos)
	if err != nil {
		return err
	}
	return nil
}

func installIntoExisting(repos []bootstrap.Repo, packages bootstrap.Package, rootfs string) (string, error) {
	//Write repos to /etc/yum.repos.d/gogo-imgbuild.repo
	if len(repos) > 0 {
		err := rpm.WriteRepos(rootfs, repos)
		if err != nil {
			return "", err
		}
	}

	//Install using builtin package manager
	//TODO auto discover the package manger
	install_cmd := "dnf install" + strings.Join(packages.Raw, " ")
	return install_cmd, nil
}

func openStore() (storage.Store, error) {
	opts, err := storage.DefaultStoreOptions()
	if err != nil {
		log.Fatalf("default store opts: %v", err)
		return nil, err
	}

	opts.GraphRoot = "/home/builder/.local/share/containers/storage"
	opts.RunRoot = "/var/tmp/storage-run-1000/containers"
	opts.GraphDriverName = "overlay"
	opts.RootlessStoragePath = ""

	return storage.GetStore(opts)
}
