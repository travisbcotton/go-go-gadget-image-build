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
    Arch   string
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
    BaseURL    string
}

func NewRepodataResolver(repos []bootstrap.Repo, arch string) *RepodataResolver {
    return &RepodataResolver{
        Client: &http.Client{Timeout: 45 * time.Second},
        Repos:  repos,
        Arch: arch,
    }
}

func (r *RepodataResolver) Resolve(pkgs bootstrap.Package) ([]bootstrap.Match, error) {
    var best_matches []bootstrap.Match
    // process packages in URL format
    for _, s := range pkgs.Raw {
        if isURL(s) && strings.HasSuffix(s, ".rpm") {
            f := path.Base(s)
            n, arch := parseNevrArchFromFilename(f)
            best_matches = append(best_matches, bootstrap.Match{
                Name: n.name, 
                EVR: evrString(n), 
                Arch: arch, 
                URL: s, 
                File: f,
            })
        }
    }
    // search through all repos to get repodata.xml metadata, compile list of packages
    var compiled_entries []entry
    for _, repo := range r.Repos {
        primaryURL, err := r.findPrimaryXML(repo.BaseURL)
        if err != nil { 
            fmt.Println("Unable to find Primary for repo ", repo.BaseURL)
            fmt.Println(err)
            continue 
        }
        baseurl := repo.BaseURL
        entries, err := r.loadPrimary(primaryURL, baseurl)
        if err != nil { 
            fmt.Println(err)
            continue 
        }
        compiled_entries = append(compiled_entries, entries...)
    }
    //Find best match in all compiled entries
    for _, s := range pkgs.Raw {
        best, err := r.findBest(compiled_entries, s)
        if err != nil {
            fmt.Println("package ", s, "not found")
        }
        best_matches = append(best_matches, best)
    }
    return best_matches, nil
}

// Search a repo for the primary.xml file and return the URL to it
func (r *RepodataResolver) findPrimaryXML(base string) (string, error) {
    url := strings.TrimRight(base, "/") + "/repodata/repomd.xml"
    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        return "", err
    }
    req.Header.Set("Accept", "application/xml, text/xml;q=0.9, */*;q=0.8")
    req.Header.Set("User-Agent", "bootstrapper/0.1 (+https://your.project)")
    resp, err := r.Client.Do(req); if err != nil { 
        fmt.Println("Error reading repomd.xml")
        return "", err 
    }
    defer resp.Body.Close()
    if resp.StatusCode != 200 { 
        return "", fmt.Errorf("%s: %s", url, resp.Status) 
    }
    var md repomd
    if err := xml.NewDecoder(resp.Body).Decode(&md); err != nil { return "", err }
    for _, d := range md.Data {
        if d.Type == "primary" {
            fmt.Println("Found primary xml:", strings.TrimRight(base, "/") + "/" + strings.TrimLeft(d.Location.Href, "/"))
            return strings.TrimRight(base, "/") + "/" + strings.TrimLeft(d.Location.Href, "/"), nil
        }
    }
    return "", errors.New("primary.xml not found")
}

// Process a primary.xml and return a list of entry structs for each package
func (r *RepodataResolver) loadPrimary(primURL string, baseurl string) ([]entry, error) {
    req, err := http.NewRequest("GET", primURL, nil)
    if err != nil {
        return nil, err
    }
    req.Header.Set("Accept", "application/xml, text/xml;q=0.9, */*;q=0.8")
    req.Header.Set("User-Agent", "bootstrapper/0.1 (+https://your.project)")
    resp, err := r.Client.Do(req); if err != nil { return nil, err }
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
            BaseURL: baseurl,
        })
    }
    return out, nil
}

// Find the best match for a package in a list of entry structs
func (r *RepodataResolver) findBest(entries []entry, pkg string) (bootstrap.Match, error){
    var best *bootstrap.Match
    for _, e := range entries {
        if strings.HasSuffix(pkg, ".rpm") {
            ok, _ := path.Match(pkg, path.Base(e.Href))
            if !ok { continue }
        } else {
            if e.Name != pkg { continue }
        }
        m := bootstrap.Match{
            Name: e.Name,
            EVR:  fmt.Sprintf("%d:%s-%s", e.Epoch, e.Ver, e.Rel),
            Arch: e.Arch,
            Href: e.Href,
            URL:  strings.TrimRight(e.BaseURL, "/") + "/" + strings.TrimLeft(e.Href, "/"),
            File: path.Base(e.Href),
        }
        if rpmEVRBetter(m, best) {
            if best != nil {
                if m.Arch != best.Arch { continue }
            }
            cp := m; best = &cp
        }
    }
    if best == nil { return bootstrap.Match{}, ErrNotFound }
    return *best, nil
}

// check if a string is a URL
func isURL(s string) bool { 
    return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") 
}

// ChatGPT assisted from here and below
func parseNevrArchFromFilename(fn string) (n nevr, arch string) {
    fn = strings.TrimSuffix(fn, ".rpm")
    parts := strings.Split(fn, ".")
    if len(parts) >= 2 { arch = parts[len(parts)-1]; fn = strings.Join(parts[:len(parts)-1], ".") }
    i := strings.LastIndex(fn, "-")
    j := strings.LastIndex(fn[:i], "-")
    if i < 0 || j < 0 { return nevr{name: fn}, arch }
    name := fn[:j]
    ver := fn[j+1 : i]
    rel := fn[i+1:]
    return nevr{name: name, ver: ver, rel: rel}, arch
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
    ia, ib := 0, 0
    for ia < len(a) || ib < len(b) {
        for ia < len(a) && !isalnum(a[ia]) { ia++ }
        for ib < len(b) && !isalnum(b[ib]) { ib++ }
        if ia >= len(a) && ib >= len(b) { return 0 }
        sa := readRun(a, &ia)
        sb := readRun(b, &ib)
        da := isdigitStr(sa)
        db := isdigitStr(sb)
        switch {
        case da && db:
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