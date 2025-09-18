package rpm

import (
	"fmt"
	"os"
	"path/filepath"
)

type OSRelease struct {
	Name        string
	ID          string
	VersionID   string
	PrettyName  string
	Version     string
}

func Write(rootfs string, r OSRelease) error {
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