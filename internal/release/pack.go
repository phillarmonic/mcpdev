package release

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/phillarmonic/mcpdev/internal/model"
	"github.com/phillarmonic/mcpdev/internal/platform"
)

const (
	modulePath     = "github.com/phillarmonic/mcpdev"
	maxMemberBytes = 64 << 20
)

// Target is one GOOS/GOARCH pair for a release archive.
type Target struct {
	GOOS   string
	GOARCH string
}

// DefaultTargets are the supported release triples.
func DefaultTargets() []Target {
	return []Target{
		{GOOS: "linux", GOARCH: "amd64"},
		{GOOS: "linux", GOARCH: "arm64"},
		{GOOS: "darwin", GOARCH: "amd64"},
		{GOOS: "darwin", GOARCH: "arm64"},
		{GOOS: "windows", GOARCH: "amd64"},
		{GOOS: "windows", GOARCH: "arm64"},
	}
}

// Options control pack output.
type Options struct {
	Root      string
	OutDir    string
	Version   string
	Commit    string
	BuildTime string
	Policy    string
	Epoch     int64
	Smoke     bool
}

// ArchiveName is mcpdev_<version>_<os>_<arch>.tar.gz or .zip on Windows.
func ArchiveName(ver, goos, goarch string) string {
	ext := ".tar.gz"
	if goos == "windows" {
		ext = ".zip"
	}
	return "mcpdev_" + sanitize(ver) + "_" + goos + "_" + goarch + ext
}

func sanitize(ver string) string {
	ver = strings.TrimPrefix(ver, "v")
	ver = strings.ReplaceAll(ver, "/", "-")
	if ver == "" {
		return "0.0.0-dev"
	}
	return ver
}

func ldflags(opt Options) string {
	return strings.Join([]string{
		"-s",
		"-w",
		"-X", modulePath + "/internal/version.Version=" + opt.Version,
		"-X", modulePath + "/internal/version.Commit=" + opt.Commit,
		"-X", modulePath + "/internal/version.BuildTime=" + opt.BuildTime,
		"-X", modulePath + "/internal/version.BuildTimePolicy=" + opt.Policy,
	}, " ")
}

// Pack cross-compiles archives and SHA256SUMS into OutDir.
func Pack(opt Options) error {
	if opt.Root == "" || opt.OutDir == "" {
		return errors.New("release: root and out dir are required")
	}
	if err := os.MkdirAll(opt.OutDir, 0o750); err != nil {
		return err
	}
	modTime := time.Unix(opt.Epoch, 0).UTC()
	var sums []string
	for _, t := range DefaultTargets() {
		name := ArchiveName(opt.Version, t.GOOS, t.GOARCH)
		binName := platform.ExecutableNameFor("mcpdev", t.GOOS)
		staging := filepath.Join(opt.OutDir, binName+"."+t.GOOS+"-"+t.GOARCH)
		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags(opt), "-o", staging, "./cmd/mcpdev") //nolint:gosec // G204: go build of this module with stamped ldflags
		cmd.Dir = opt.Root
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS="+t.GOOS,
			"GOARCH="+t.GOARCH,
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("release: build %s/%s: %w\n%s", t.GOOS, t.GOARCH, err, out)
		}
		archive := filepath.Join(opt.OutDir, name)
		if t.GOOS == "windows" {
			err = writeZip(archive, binName, staging, modTime)
		} else {
			err = writeTarGz(archive, binName, staging, modTime)
		}
		_ = os.Remove(staging)
		if err != nil {
			return err
		}
		sum, sumErr := fileSHA256(archive)
		if sumErr != nil {
			return sumErr
		}
		sums = append(sums, sum+"  "+name)
	}
	sort.Strings(sums)
	body := strings.Join(sums, "\n") + "\n"
	if err := os.WriteFile(filepath.Join(opt.OutDir, "SHA256SUMS"), []byte(body), 0o600); err != nil {
		return err
	}
	if !opt.Smoke {
		return nil
	}
	return SmokeHost(opt)
}

func writeTarGz(archive, name, src string, modTime time.Time) error {
	f, err := os.Create(archive) //nolint:gosec // G304: path is dist output we control
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gw := gzip.NewWriter(f)
	gw.ModTime = modTime
	tw := tar.NewWriter(gw)
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	hdr, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	hdr.Name = name
	hdr.Uid = 0
	hdr.Gid = 0
	hdr.Uname = ""
	hdr.Gname = ""
	hdr.ModTime = modTime
	hdr.AccessTime = time.Time{}
	hdr.ChangeTime = time.Time{}
	hdr.Mode = 0o755
	headerErr := tw.WriteHeader(hdr)
	if headerErr != nil {
		return headerErr
	}
	in, err := os.Open(src) //nolint:gosec // G304: staging binary we just built
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if copyErr := copyLimited(tw, in); copyErr != nil {
		return copyErr
	}
	if closeErr := tw.Close(); closeErr != nil {
		return closeErr
	}
	return gw.Close()
}

func writeZip(archive, name, src string, modTime time.Time) error {
	f, err := os.Create(archive) //nolint:gosec // G304: path is dist output we control
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	zw := zip.NewWriter(f)
	hdr := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: modTime}
	hdr.SetMode(0o755)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	in, err := os.Open(src) //nolint:gosec // G304: staging binary we just built
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	if copyErr := copyLimited(w, in); copyErr != nil {
		return copyErr
	}
	return zw.Close()
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path) //nolint:gosec // G304: dist archive we wrote
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if copyErr := copyLimited(h, f); copyErr != nil {
		return "", copyErr
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func copyLimited(dst io.Writer, src io.Reader) error {
	n, err := io.CopyN(dst, src, maxMemberBytes+1)
	if n > maxMemberBytes {
		return errors.New("release: member exceeds 64MiB bound")
	}
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	return nil
}

// SmokeHost unpacks the host triple archive and runs version plus a stdio inspect.
func SmokeHost(opt Options) error {
	name := ArchiveName(opt.Version, runtime.GOOS, runtime.GOARCH)
	archive := filepath.Join(opt.OutDir, name)
	dir := filepath.Join(opt.OutDir, "smoke")
	_ = os.RemoveAll(dir)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	binName := platform.ExecutableName("mcpdev")
	var extractErr error
	if runtime.GOOS == "windows" {
		extractErr = unzipFile(archive, dir, binName)
	} else {
		extractErr = untarFile(archive, dir, binName)
	}
	if extractErr != nil {
		return extractErr
	}
	bin := filepath.Join(dir, binName)
	verOut, err := exec.Command(bin, "--format", "json", "version").CombinedOutput() //nolint:gosec // G204: binary we just packed
	if err != nil {
		return fmt.Errorf("release: version smoke: %w\n%s", err, verOut)
	}
	if !strings.Contains(string(verOut), `"snapshot_schema":`+strconv.Itoa(model.SnapshotSchemaVersion)) {
		return fmt.Errorf("release: version smoke missing snapshot_schema: %s", verOut)
	}
	if !strings.Contains(string(verOut), `"commit":"`+opt.Commit) {
		return fmt.Errorf("release: version smoke missing commit: %s", verOut)
	}
	lab := filepath.Join(dir, platform.ExecutableName("stdio-lab"))
	build := exec.Command("go", "build", "-o", lab, ".") //nolint:gosec // G204: fixed testdata/servers/stdio-lab
	build.Dir = filepath.Join(opt.Root, "testdata", "servers", "stdio-lab")
	out, err := build.CombinedOutput()
	if err != nil {
		return fmt.Errorf("release: build stdio-lab: %w\n%s", err, out)
	}
	ins := exec.Command(bin, "--format", "json", "--command", lab, "inspect", "tools") //nolint:gosec // G204: packed mcpdev and lab binary
	ins.Env = append(os.Environ(), "MCPDEV_FIXTURE=legacy-stdio")
	insOut, err := ins.CombinedOutput()
	if err != nil {
		return fmt.Errorf("release: inspect smoke: %w\n%s", err, insOut)
	}
	if !strings.Contains(string(insOut), `"name":"echo"`) {
		return fmt.Errorf("release: inspect smoke missing echo tool: %s", insOut)
	}
	return nil
}

func unzipFile(archive, dir, want string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return err
	}
	defer func() { _ = r.Close() }()
	for _, f := range r.File {
		if f.Name != want {
			continue
		}
		return extractZipMember(f, filepath.Join(dir, want))
	}
	return fmt.Errorf("release: %s not in zip", want)
}

func extractZipMember(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer func() { _ = rc.Close() }()
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o750) //nolint:gosec // G304: smoke dir we created
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	return copyLimited(out, rc)
}

func untarFile(archive, dir, want string) error {
	f, err := os.Open(archive) //nolint:gosec // G304: dist archive
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer func() { _ = gr.Close() }()
	tr := tar.NewReader(gr)
	for {
		hdr, nextErr := tr.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return nextErr
		}
		if hdr.Name != want {
			continue
		}
		out, openErr := os.OpenFile(filepath.Join(dir, want), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o750) //nolint:gosec // G304: smoke dir
		if openErr != nil {
			return openErr
		}
		copyErr := copyLimited(out, tr)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	}
	return fmt.Errorf("release: %s not in archive", want)
}
