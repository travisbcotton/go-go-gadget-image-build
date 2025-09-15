package bootstrap

type Repo struct {
    BaseURL string
    Arch    string
}

type Spec struct {
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
    Resolve(spec Spec) (Match, error)
}