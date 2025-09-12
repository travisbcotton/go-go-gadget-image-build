package format

type FetchRequest struct {
    Repos    []string
    Packages []string
    Arch     string
}

type OrderedPackage struct {
    Path     string
    Name     string
    Provides []string
    Requires []string
}

type Bootstrap interface {
    Name() string
    Fetch(req FetchRequest) ([]string, error)
    Analyze(paths []string) ([]OrderedPackage, []string /*unresolved*/, error)
    Extract(rootfs string, ordered []OrderedPackage) error
    RegisterDB(rootfs string, ordered []OrderedPackage) error
}