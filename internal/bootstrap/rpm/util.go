package rpm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"errors"

	"github.com/travisbcotton/go-go-gadget-image-build/pkg/bootstrap"
)

type OSRelease struct {
	Name        string
	ID          string
	VersionID   string
	PrettyName  string
	Version     string
}

func WriteOSRelease(rootfs string, r OSRelease) error {
	etcDir := filepath.Join(rootfs, "etc")
	if err := os.MkdirAll(etcDir, 0o755); err != nil {
		return err
	}

	// Build minimal content; quote values as per spec.
	pretty := r.PrettyName
	if pretty == "" {
		pretty = r.Name + " " + r.VersionID
	}
	content := fmt.Sprintf(
		"NAME=%q\nID=%s\nVERSION_ID=%q\nPRETTY_NAME=%q\n",
		r.Name, r.ID, r.VersionID, pretty,
	)
	if r.Version != "" {
		content += fmt.Sprintf("VERSION=%q\n", r.Version)
	}

	etcPath := filepath.Join(etcDir, "os-release")
	if err := os.WriteFile(etcPath, []byte(content), 0o644); err != nil {
		return err
	}

	// Many distros have /etc/os-release -> /usr/lib/os-release or vice versa.
	// To be robust, ensure /usr/lib exists and create a symlink there pointing to /etc/os-release.
	usrLib := filepath.Join(rootfs, "usr", "lib")
	if err := os.MkdirAll(usrLib, 0o755); err != nil {
		return err
	}
	libPath := filepath.Join(usrLib, "os-release")
	_ = os.Remove(libPath) // replace if exists
	if err := os.Symlink("/etc/os-release", libPath); err != nil {
		// If symlinks aren’t desired, you could copy instead:
		// _ = copyFile(etcPath, libPath)
		// but usually the symlink is fine.
		return err
	}
	return nil
}

func WriteRepos(rootfs string, repos []bootstrap.Repo) error {
	if len(repos) == 0 {
		return errors.New("no repos to write")
	}
	dir := filepath.Join(rootfs, "etc", "yum.repos.d")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	filename := "gogo-imgbuild.repo"
	var b strings.Builder
	for _, r := range repos {
		if r.ID == "" {
			return errors.New("repo.ID is required")
		}
		fmt.Fprintf(&b, "[%s]\n", r.ID)
		fmt.Fprintf(&b, "name=%s\n", escape(r.ID))
		fmt.Fprintf(&b, "baseurl=%s\n", strings.Join(r.BaseURL, ","))
		fmt.Fprintf(&b, "enabled=1\n")
	}
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}