package main

import (
    "fmt"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/bootstrap/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

func main() {
    driver := rpm.New()
    pkgs := []string{"pkg1","pkg2"}
    repos := []string{"http://repo1", "http://repo2"}

    req := format.FetchRequest{
        Repos:    repos,
        Packages: pkgs,
    }
    driver.Fetch(req)
    fmt.Println("Driver name:", driver.Name())
}
