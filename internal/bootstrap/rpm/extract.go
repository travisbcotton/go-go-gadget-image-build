package rpm

import (
	"io"
	"path/filepath"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"bufio"

	"github.com/cavaliergopher/cpio"
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
		fmt.Println("filename?: ", name)
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
		err := ExtractRPM(r, dest)
		if err != nil {
			return fmt.Errorf("extract %s: %w", r ,err)
		}
	}
	return nil
}