package main

import (
    "fmt"

    "github.com/yourname/bootstrapper/internal/drivers/rpm"
)

func main() {
    driver := rpm.New()
    fmt.Println("Driver name:", driver.Name())
    fmt.Println(driver.Hello())
}
