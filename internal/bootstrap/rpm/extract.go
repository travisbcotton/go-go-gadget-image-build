package rpm

import (
	"io"
	"path/filepath"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"bufio"
	"strings"

	"github.com/cavaliergopher/cpio"
)

const (
    cpioModeTypeMask = 0170000
    cpioModeDir      = 0040000
    cpioModeReg      = 0100000
    cpioModeSymlink  = 0120000
    cpioModeChar     = 0020000
    cpioModeBlock    = 0060000
    cpioModeFIFO     = 0010000
    cpioModeSocket   = 0140000
)

func ExtractCPIOStream(r io.Reader, dest string) error {
	dest = filepath.Clean(dest)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	br := bufio.NewReader(r)
	cr := cpio.NewReader(br)

	for {
		header, err := cr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("cpio read header: %w", err)
		}
		name := header.Name

		for strings.HasPrefix(name, "./") {
			name = strings.TrimPrefix(name, "./")
		}
		name = strings.TrimLeft(name,"/")

		target := filepath.Join(dest, filepath.Clean(name))
		if !strings.HasPrefix(target, dest+string(os.PathSeparator)) && target != dest {
			if _, err := io.Copy(io.Discard, cr); err != nil {
				return err
			}
			continue
		}
		mode := os.FileMode(header.Mode)
		mt := header.ModTime

		switch header.Mode & cpioModeTypeMask {
		case cpio.TypeDir:
			fmt.Println("file ", name, " is a directory")
			if err := os.MkdirAll(target, mode|0o111); err != nil {
				return err
			}
			_ = os.Chtimes(target, mt, mt)
		case cpio.TypeReg:
			fmt.Println("file ", name, " is a regular file")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, cr); err != nil {
				_ = f.Close()
				return err
			}
			if err := f.Close(); err != nil {
				return err
			}
			_ = os.Chtimes(target, mt, mt)
		case cpio.TypeSymlink:
			fmt.Println("file ", name, " is a symlink")
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			if err := os.Symlink(header.Linkname, target); err != nil {
				return err
			}
		default:
			fmt.Println("file ", name, " is an unknown type")
			if _, err := io.Copy(io.Discard, cr); err != nil {
				return err
			}
		}
	}
}

func ExtractRPM(rpm_path string, dest string) error {
	cmd := exec.Command("rpm2cpio", rpm_path)
	cmd.Stderr = os.Stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	defer func() { _ = cmd.Wait() }()
	return ExtractCPIOStream(stdout, filepath.Clean(dest))
}

func InstallRPMs(rpms []string, dest string) error {
	for _, r := range rpms {
		fmt.Println("Extracting ", r)
		err := ExtractRPM(r, dest)
		if err != nil {
			return fmt.Errorf("extract %s: %w", r ,err)
		}
		fmt.Println("Finished ", r)
	}
	return nil
}