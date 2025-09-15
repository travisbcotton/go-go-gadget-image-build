package rpm

import (
    getter "github.com/hashicorp/go-getter"
    "net/http"
    "context"
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
    c := *d.Client // shallow copy so we can tweak per-call
    c.Ctx = ctx
    c.Src = url
    c.Dst = dest
    // Optional: set c.ProgressTracker, c.Pwd, c.Insecure, c.Umask, etc.

    if err := c.Get(); err != nil {
        return DownloadResult{}, err
    }
    return DownloadResult{Path: dest}, nil
}