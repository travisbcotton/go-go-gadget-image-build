package rpm

import "github.com/yourname/bootstrapper/pkg/format"

type RPMDriver struct{}

func New() format.Driver {
    return &RPMDriver{}
}

func (d *RPMDriver) Name() string {
    return "rpm"
}

func (d *RPMDriver) Hello() string {
    return "Hello from RPM driver!"
}
