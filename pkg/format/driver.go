package format

type Driver interface {
    Name() string
    Hello() string
}
