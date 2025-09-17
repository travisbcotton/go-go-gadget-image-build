package rpm

import (
    getter "github.com/hashicorp/go-getter"
    "net/http"
    "context"
    "path"
    "path/filepath"
    "net/url"
    "strings"
)

type GetterDownloader struct {
    Client *getter.Client
}

type DownloadResult struct {
	Path         string
}

func NewGetterDownloader(hc *http.Client) *GetterDownloader {
    getters := map[string]getter.Getter{
        "http":  &getter.HttpGetter{Client: hc},
        "https": &getter.HttpGetter{Client: hc},
    }
    return &GetterDownloader{
        Client: &getter.Client{
            Getters: getters,
            Mode: getter.ClientModeAny,
        },
    }
}

func (d *GetterDownloader) DownloadRPM(ctx context.Context, url string, dest string) (DownloadResult, error) {
    filename := filenameFromURL(url)
    rpm_path := filepath.Join(dest, filename)

    c := *d.Client
    c.Ctx = ctx
    c.Src = url
    c.Dst = rpm_path
    c.Mode = getter.ClientModeFile

    if err := c.Get(); err != nil {
        return DownloadResult{}, err
    }
    return DownloadResult{Path: rpm_path}, nil
}

func filenameFromURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "download.rpm"
	}
	base := path.Base(u.Path)
	if base == "" || base == "." || base == "/" {
		return "download.rpm"
	}
	// strip query noise, if any (rare for RPMs)
	if i := strings.IndexByte(base, '?'); i >= 0 {
		base = base[:i]
	}
	return base
}