package rpm

import (
    "fmt"
    "time"
    "errors"
    "net/http"
    "strings"
    "path"
    "encoding/xml"
    "compress/gzip"
    "unicode"
    "github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

var ErrNotFound = errors.New("no match")

type nevr struct{ epoch int; ver, rel, name string }

type repomd struct {
    XMLName xml.Name   `xml:"repomd"`
    Data    []repodata `xml:"data"`
}
type repodata struct {
    Type     string   `xml:"type,attr"`
    Location location `xml:"location"`
}
type location struct {
    Href string `xml:"href,attr"`
}

type RepodataResolver struct {
    Client *http.Client
    Repos  []bootstrap.Repo
}

type primary struct {
    XMLName  xml.Name     `xml:"metadata"`
    Packages []primaryPkg `xml:"package"`
}
type primaryPkg struct {
    Name     string       `xml:"name"`
    Arch     string       `xml:"arch"`
    Version  primaryVer   `xml:"version"`
    Location location     `xml:"location"`
}
type primaryVer struct {
    Epoch int    `xml:"epoch,attr"`
    Ver   string `xml:"ver,attr"`
    Rel   string `xml:"rel,attr"`
}

type entry struct {
    Name       string
    Arch       string
    Epoch      int
    Ver, Rel   string
    Href       string
}

func NewRepodataResolver(repos []bootstrap.Repo) *RepodataResolver {
    return &RepodataResolver{
        Client: &http.Client{Timeout: 45 * time.Second},
        Repos:  repos,
    }
}

func (r *RepodataResolver) Resolve(s bootstrap.Spec) (bootstrap.Match, error) {
    if isURL(s.Raw) && strings.HasSuffix(s.Raw, ".rpm") {
        f := path.Base(s.Raw)
        n, arch := parseNevrArchFromFilename(f)
        return bootstrap.Match{Name: n.name, EVR: evrString(n), Arch: arch, URL: s.Raw, File: f}, nil
    }

    var best *bootstrap.Match
    for _, repo := range r.Repos {
        primaryURL, err := r.findPrimaryXML(repo.BaseURL)
        if err != nil { continue }
        entries, err := r.loadPrimary(primaryURL)
        if err != nil { continue }

        for _, e := range entries {
            if repo.Arch != "" && e.Arch != repo.Arch { continue }
            if strings.HasSuffix(s.Raw, ".rpm") {
                ok, _ := path.Match(s.Raw, path.Base(e.Href))
                if !ok { continue }
            } else {
                if e.Name != s.Raw { continue }
            }
            m := bootstrap.Match{
                Name: e.Name,
                EVR:  fmt.Sprintf("%d:%s-%s", e.Epoch, e.Ver, e.Rel),
                Arch: e.Arch,
                Href: e.Href,
                URL:  strings.TrimRight(repo.BaseURL, "/") + "/" + strings.TrimLeft(e.Href, "/"),
                File: path.Base(e.Href),
            }
            if rpmEVRBetter(m, best) {
                cp := m; best = &cp
            }
        }
    }
    if best == nil { return bootstrap.Match{}, ErrNotFound }
    return *best, nil
}

func isURL(s string) bool { 
    return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") 
}

func parseNevrArchFromFilename(fn string) (n nevr, arch string) {
    // Strip .rpm
    fn = strings.TrimSuffix(fn, ".rpm")
    parts := strings.Split(fn, ".")
    if len(parts) >= 2 { arch = parts[len(parts)-1]; fn = strings.Join(parts[:len(parts)-1], ".") }
    // split name-ver-rel (name may contain dashes; take last 2 dashes)
    i := strings.LastIndex(fn, "-")
    j := strings.LastIndex(fn[:i], "-")
    if i < 0 || j < 0 { return nevr{name: fn}, arch }
    name := fn[:j]
    ver := fn[j+1 : i]
    rel := fn[i+1:]
    return nevr{name: name, ver: ver, rel: rel}, arch
}

func (r *RepodataResolver) findPrimaryXML(base string) (string, error) {
    url := strings.TrimRight(base, "/") + "/repodata/repomd.xml"
    resp, err := r.Client.Get(url); if err != nil { return "", err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 { return "", fmt.Errorf("%s: %s", url, resp.Status) }
    var md repomd
    if err := xml.NewDecoder(resp.Body).Decode(&md); err != nil { return "", err }
    for _, d := range md.Data {
        if d.Type == "primary" {
            return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(d.Location.Href, "/"), nil
        }
    }
    return "", errors.New("primary.xml not found")
}

func (r *RepodataResolver) loadPrimary(primURL string) ([]entry, error) {
    resp, err := r.Client.Get(primURL); if err != nil { return nil, err }
    defer resp.Body.Close()
    if resp.StatusCode != 200 { return nil, fmt.Errorf("%s: %s", primURL, resp.Status) }
    zr, err := gzip.NewReader(resp.Body); if err != nil { return nil, err }
    defer zr.Close()
    var meta primary
    if err := xml.NewDecoder(zr).Decode(&meta); err != nil { return nil, err }
    out := make([]entry, 0, len(meta.Packages))
    for _, p := range meta.Packages {
        out = append(out, entry{
            Name:  p.Name,
            Arch:  p.Arch,
            Epoch: p.Version.Epoch,
            Ver:   p.Version.Ver,
            Rel:   p.Version.Rel,
            Href:  p.Location.Href,
        })
    }
    return out, nil
}

func evrString(n nevr) string {
    if n.epoch > 0 { return fmt.Sprintf("%d:%s-%s", n.epoch, n.ver, n.rel) }
    return fmt.Sprintf("%s-%s", n.ver, n.rel)
}

func rpmEVRBetter(a bootstrap.Match, cur *bootstrap.Match) bool {
    if cur == nil { return true }
    ae, av, ar := splitEVR(a.EVR)
    ce, cv, cr := splitEVR(cur.EVR)
    if ae != ce { return ae > ce }
    if c := rpmvercmp(av, cv); c != 0 { return c > 0 }
    return rpmvercmp(ar, cr) > 0
}

func splitEVR(evr string) (epoch int, ver, rel string) {
    // evr is like "0:2.34-100.el9" or "2.34-100.el9"
    epoch = 0
    s := evr
    if i := indexByte(s, ':'); i >= 0 {
        epoch = atoiSafe(s[:i])
        s = s[i+1:]
    }
    if j := indexByte(s, '-'); j >= 0 {
        ver, rel = s[:j], s[j+1:]
    } else {
        ver = s
    }
    return
}

func rpmvercmp(a, b string) int {
    // Rough port of rpmdev-vercmp logic: split into alnum runs
    ia, ib := 0, 0
    for ia < len(a) || ib < len(b) {
        // skip non-alnum
        for ia < len(a) && !isalnum(a[ia]) { ia++ }
        for ib < len(b) && !isalnum(b[ib]) { ib++ }
        if ia >= len(a) && ib >= len(b) { return 0 }
        // grab sub
        sa := readRun(a, &ia)
        sb := readRun(b, &ib)
        da := isdigitStr(sa)
        db := isdigitStr(sb)
        switch {
        case da && db:
            // trim leading zeros
            for len(sa) > 0 && sa[0] == '0' { sa = sa[1:] }
            for len(sb) > 0 && sb[0] == '0' { sb = sb[1:] }
            if len(sa) != len(sb) {
                if len(sa) > len(sb) { return 1 } else { return -1 }
            }
            if sa > sb { return 1 }
            if sa < sb { return -1 }
        case !da && !db:
            if sa > sb { return 1 }
            if sa < sb { return -1 }
        default:
            // numeric is greater than alpha
            if da { return 1 } else { return -1 }
        }
    }
    return 0
}


func readRun(s string, i *int) string {
    if *i >= len(s) { return "" }
    j := *i
    isNum := unicode.IsDigit(rune(s[j]))
    for j < len(s) && unicode.IsLetter(rune(s[j])) == !isNum && unicode.IsDigit(rune(s[j])) == isNum {
        j++
    }
    out := s[*i:j]
    *i = j
    return out
}

func isdigitStr(s string) bool {
    if s == "" { return false }
    for i := 0; i < len(s); i++ {
        if s[i] < '0' || s[i] > '9' { return false }
    }
    return true
}

func indexByte(s string, c byte) int {
    for i := 0; i < len(s); i++ { if s[i] == c { return i } }
    return -1
}

func atoiSafe(s string) int {
    n := 0
    for i := 0; i < len(s); i++ { if s[i] >= '0' && s[i] <= '9' { n = n*10 + int(s[i]-'0') } }
    return n
}

func isalnum(b byte) bool {
    r := rune(b)
    return unicode.IsLetter(r) || unicode.IsDigit(r)
}