package main

import (
    "fmt"

    "github.com/travisbcotton/go-go-gadget-image-build/internal/drivers/rpm"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/format"
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
