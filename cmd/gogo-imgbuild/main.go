package main

/*

 */
import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/containers/buildah"
	"github.com/containers/buildah/define"
	storageRef "github.com/containers/image/v5/storage"
	"github.com/containers/storage/pkg/reexec"

	"github.com/travisbcotton/go-go-gadget-image-build/internal/config"
	"github.com/travisbcotton/go-go-gadget-image-build/internal/layer"
)

func main() {
	if reexec.Init() {
		return
	}
	defaultCfg := "./bootstrap.yaml"
	cfgPath := flag.String("config", defaultCfg, "path to YAML config (or '-' for stdin)")
	flag.Parse()

	// load config file
	cfg, err := config.Load(*cfgPath)
	if err != nil {
		log.Fatal(err)
	}

	bctx := context.Background()

	builder, err := layer.NewLayer(cfg.Opts.Name, cfg.Opts.Parent)
	if err != nil {
		log.Fatalf("new builder: %v", err)
	}
	defer func() { _ = builder.Builder.Delete() }()
	defer builder.Store.Shutdown(false)

	err = builder.BuildLayer(*cfg)
	if err != nil {
		log.Fatalf("Failed to build layer %v", err)
		panic(err)
	}

	imageName := cfg.Opts.Name
	dest, err := storageRef.Transport.ParseStoreReference(builder.Store, imageName)
	if err != nil {
		panic(err)
	}
	fmt.Println("Committing Container")
	_, _, _, err = builder.Builder.Commit(bctx, dest, buildah.CommitOptions{
		PreferredManifestType: define.OCIv1ImageManifest,
	})
	if err != nil {
		log.Fatalf("commit: %v", err)
	}
	fmt.Println("Committed image:", imageName)
}
