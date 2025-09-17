package bootstrap

type Repo struct {
    BaseURL string
}

type Package struct {
    Raw []string
}

type Match struct {
    Name   string
    EVR    string
    Arch   string
    Href   string
    URL    string
    File   string
}

type Resolver interface {
    Resolve(pkgs Package) (Match, error)
}