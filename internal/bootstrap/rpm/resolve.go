package rpm

import (
    "fmt"
    "time"
    "errors"
    "net/http"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

type RPM struct {
    http *http.Client
}

func New() *RPM {
    return &RPM{
        http: &http.Client{Timeout: 30 * time.Second},
    }
}

func (d *RPM) Name() string { return "rpm" }

func (d *RPM) Fetch(req format.FetchRequest) ([]string, error) {
    if len(req.Packages) == 0 {
        return nil, errors.New("no packages provided")
    }
    if len(req.Repos) == 0 {
	return nil, errors.New("No repos provided")
    }
    fmt.Println("Repos:", req.Repos)
    fmt.Println("Packages:", req.Packages)
    return nil,nil
}
