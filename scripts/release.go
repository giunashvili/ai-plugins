// Run from the ai-plugins root: go run ./scripts/release.go select
// This prepares local release files only. It never commits, pushes or uploads.
package main

import (
	"archive/zip"
	"crypto/sha256"
	"debug/buildinfo"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

var releaseTargets = []string{"darwin/amd64", "darwin/arm64", "linux/amd64", "linux/arm64", "windows/amd64", "windows/arm64"}

var packagePaths = []string{"plugin.json", ".codex-plugin", ".claude-plugin", "mcp.json", ".mcp.json", "skills", "assets", "README.md", "THIRD_PARTY_NOTICES.txt", "docs/architecture.md", "docs/maintenance.md", "docs/team-distribution.md", "docs/extension-setup.md", "docs/references.md"}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Release:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) == 0 || args[0] != "select" {
		return errors.New("usage: go run ./scripts/release.go select [-check] [-binaries /directory]")
	}
	flags := flag.NewFlagSet("select", flag.ContinueOnError)
	checkOnly := flags.Bool("check", false, "validate source and show the release plan without building or writing packages")
	prebuilt := flags.String("binaries", "", "explicitly reuse six existing GUI builds from this directory instead of rebuilding")
	if err := flags.Parse(args[1:]); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	root, err := os.Getwd()
	if err != nil {
		return err
	}
	source := filepath.Join(root, "plugins", "select")
	version, err := sourceVersion(source)
	if err != nil {
		return err
	}
	if previous, err := readJSON(filepath.Join(root, "published", "select", "plugin.json")); err == nil {
		if previous["version"] == version && !*checkOnly {
			return errors.New("increment Select's version in its three manifests and src/mcp.go before the next release; archive filenames remain unversioned")
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	for _, path := range append(append([]string{}, packagePaths...), "scripts/launch/select", "launcher/main_windows.go") {
		if _, err := os.Lstat(filepath.Join(source, path)); err != nil {
			return err
		}
	}
	if err := command(source, nil, "go", "run", "./scripts", "check"); err != nil {
		return err
	}
	catalogs, err := updatedCatalogs(root, version)
	if err != nil {
		return err
	}
	fmt.Printf("Select %s: macOS, Linux and Windows; x64 and ARM64.\n", version)
	if *checkOnly {
		fmt.Println("Plan: build six GUI executables; assemble published/select; write releases/select.zip and checksums; update both catalogs. No Git commands or uploads.")
		return nil
	}
	binaries := filepath.Join(source, "build")
	if *prebuilt != "" {
		binaries, err = filepath.Abs(*prebuilt)
		if err != nil {
			return err
		}
		fmt.Println("Using explicitly supplied binaries; their source freshness must be verified by the publisher.")
	} else {
		if runtime.GOOS != "darwin" {
			return errors.New("run the full build on a Mac with Xcode tools and Docker Desktop; or supply all six GUI builds using -binaries")
		}
		if _, err := exec.LookPath("docker"); err != nil {
			return errors.New("Docker Desktop is needed for Linux GUI builds")
		}
		info := exec.Command("docker", "info", "--format", "{{.OSType}}")
		output, err := info.Output()
		if err != nil || strings.TrimSpace(string(output)) != "linux" {
			return errors.New("start Docker Desktop with Linux containers before making a release")
		}
		for _, target := range releaseTargets {
			fmt.Println("Building", target)
			if strings.HasPrefix(target, "linux/") {
				err = buildLinux(source, target)
			} else {
				err = command(source, nil, "go", "run", "./scripts", "build", "-target", target)
			}
			if err != nil {
				return fmt.Errorf("build %s: %w", target, err)
			}
		}
	}
	for _, target := range releaseTargets {
		if err := verifyBinary(binaryPath(binaries, target), target); err != nil {
			return err
		}
	}
	work, err := os.MkdirTemp("", "ai-plugins-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)
	stage := filepath.Join(work, "select")
	if err := assemble(source, stage, binaries); err != nil {
		return err
	}
	if err := command(source, []string{"GOOS=windows", "GOARCH=amd64", "CGO_ENABLED=0"}, "go", "build", "-trimpath", "-ldflags", "-s -w -H windowsgui", "-o", filepath.Join(stage, "server", "select.exe"), "./launcher"); err != nil {
		return fmt.Errorf("build Windows launcher: %w", err)
	}
	archive := filepath.Join(work, "select.zip")
	if err := writeZIP(stage, archive); err != nil {
		return err
	}
	data, err := os.ReadFile(archive)
	if err != nil {
		return err
	}
	digest := sha256.Sum256(data)
	sums := []byte(hex.EncodeToString(digest[:]) + "  select.zip\n")
	// All builds and assembly must succeed before replacing published content.
	if err := replacePackage(stage, filepath.Join(root, "published", "select")); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(root, "releases", "select.zip"), data); err != nil {
		return err
	}
	if err := atomicWrite(filepath.Join(root, "releases", "SHA256SUMS-select.txt"), sums); err != nil {
		return err
	}
	for _, name := range []string{".agents/plugins/marketplace.json", ".claude-plugin/marketplace.json"} {
		if err := atomicWrite(filepath.Join(root, name), catalogs[name]); err != nil {
			return err
		}
	}
	fmt.Println("Prepared published/select and releases/select.zip; both marketplace catalogs now list Select.")
	fmt.Println("Review the files. Commit and push only when explicitly requested. No release has been uploaded.")
	return nil
}

func sourceVersion(source string) (string, error) {
	var version string
	for _, name := range []string{"plugin.json", ".codex-plugin/plugin.json", ".claude-plugin/plugin.json"} {
		manifest, err := readJSON(filepath.Join(source, name))
		if err != nil {
			return "", fmt.Errorf("run from the ai-plugins repository root: %w", err)
		}
		value, ok := manifest["version"].(string)
		if manifest["name"] != "select" || !ok || value == "" || (version != "" && value != version) {
			return "", errors.New("Select identities or manifest versions do not match")
		}
		version = value
	}
	code, err := os.ReadFile(filepath.Join(source, "src", "mcp.go"))
	if err != nil {
		return "", err
	}
	match := regexp.MustCompile(`(?m)^const version = "([^"]+)"\r?$`).FindSubmatch(code)
	if match == nil || string(match[1]) != version {
		return "", errors.New("src/mcp.go version must match the three Select manifests")
	}
	return version, nil
}

func buildLinux(source, target string) error {
	mod, err := os.ReadFile(filepath.Join(source, "go.mod"))
	if err != nil {
		return err
	}
	match := regexp.MustCompile(`(?m)^go ([0-9]+\.[0-9]+\.[0-9]+)\r?$`).FindSubmatch(mod)
	if match == nil {
		return errors.New("go.mod must pin a Go patch version for Linux builds")
	}
	output := filepath.Join(source, "build", strings.ReplaceAll(target, "/", "-"))
	if err := os.MkdirAll(output, 0755); err != nil {
		return err
	}
	context, err := os.MkdirTemp("", "select-linux-source-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(context)
	// Copy an allowlist so Docker never receives local profiles or preview data.
	for _, name := range []string{"go.mod", "go.sum", "plugin.json", ".codex-plugin/plugin.json", ".claude-plugin/plugin.json", "mcp.json", ".mcp.json", "src", "scripts/native.go", "skills", "assets"} {
		if err := copyPath(filepath.Join(source, name), filepath.Join(context, name)); err != nil {
			return err
		}
	}
	script := `set -eu
apt-get update
apt-get install -y --no-install-recommends gcc pkg-config libgtk-3-dev libwebkit2gtk-4.1-dev
mkdir /tmp/select
cd /source
tar --exclude=./build --exclude=./releases --exclude=./.git -cf - . | tar -xf - -C /tmp/select
cd /tmp/select
go run ./scripts build -target "$1"
cp "build/$2/select" /output/select
chmod 755 /output/select`
	return command(source, nil, "docker", "run", "--rm", "--platform", target,
		"--mount", "type=bind,source="+context+",target=/source,readonly",
		"--mount", "type=bind,source="+output+",target=/output",
		"golang:"+string(match[1])+"-bookworm", "sh", "-c", script, "release-build", target, strings.ReplaceAll(target, "/", "-"))
}

func command(directory string, env []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir, cmd.Stdout, cmd.Stderr = directory, os.Stdout, os.Stderr
	cmd.Env = append(os.Environ(), env...)
	return cmd.Run()
}

func binaryPath(root, target string) string {
	name := "select"
	if strings.HasPrefix(target, "windows/") {
		name += ".exe"
	}
	return filepath.Join(root, strings.ReplaceAll(target, "/", "-"), name)
}

func verifyBinary(path, target string) error {
	info, err := buildinfo.ReadFile(path)
	if err != nil {
		return fmt.Errorf("%s: missing or invalid Go executable: %w", target, err)
	}
	settings := map[string]string{}
	for _, setting := range info.Settings {
		settings[setting.Key] = setting.Value
	}
	parts := strings.Split(target, "/")
	tags := strings.Split(settings["-tags"], ",")
	production := false
	for _, tag := range tags {
		if tag == "headless" {
			return fmt.Errorf("%s: headless builds cannot be released", target)
		}
		production = production || tag == "production"
	}
	if settings["GOOS"] != parts[0] || settings["GOARCH"] != parts[1] || !production {
		return fmt.Errorf("%s: expected a matching production GUI build", target)
	}
	return nil
}

func assemble(source, stage, binaries string) error {
	for _, name := range packagePaths {
		if err := copyPath(filepath.Join(source, name), filepath.Join(stage, name)); err != nil {
			return err
		}
	}
	for _, target := range releaseTargets {
		path := binaryPath(binaries, target)
		destination := binaryPath(filepath.Join(stage, "server"), target)
		if err := copyPath(path, destination); err != nil {
			return err
		}
		if err := os.Chmod(destination, 0755); err != nil {
			return err
		}
	}
	launcher := filepath.Join(stage, "server", "select")
	if err := copyPath(filepath.Join(source, "scripts", "launch", "select"), launcher); err != nil {
		return err
	}
	return os.Chmod(launcher, 0755)
}

func updatedCatalogs(root, version string) (map[string][]byte, error) {
	result := map[string][]byte{}
	for _, name := range []string{".agents/plugins/marketplace.json", ".claude-plugin/marketplace.json"} {
		catalog, err := readJSON(filepath.Join(root, name))
		if err != nil {
			return nil, err
		}
		entries, ok := catalog["plugins"].([]any)
		if !ok {
			return nil, fmt.Errorf("%s: plugins must be an array", name)
		}
		entry := map[string]any{"name": "select"}
		found := -1
		for i, value := range entries {
			existing, ok := value.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("%s: invalid plugin entry", name)
			}
			if existing["name"] == "select" {
				if found != -1 {
					return nil, fmt.Errorf("%s: duplicate Select entries", name)
				}
				entry, found = existing, i
			}
		}
		if strings.HasPrefix(name, ".agents/") {
			entry["source"] = map[string]string{"source": "local", "path": "./published/select"}
		} else {
			entry["source"], entry["version"] = "./published/select", version
		}
		if found == -1 {
			entries = append(entries, entry)
		} else {
			entries[found] = entry
		}
		catalog["plugins"] = entries
		data, err := json.MarshalIndent(catalog, "", "  ")
		if err != nil {
			return nil, err
		}
		result[name] = append(data, '\n')
	}
	return result, nil
}

func readJSON(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var value map[string]any
	err = json.Unmarshal(data, &value)
	return value, err
}

func copyPath(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("symlinks cannot be published: %s", path)
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("only regular package files can be published: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func writeZIP(source, destination string) error {
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer file.Close()
	archive := zip.NewWriter(file)
	var paths []string
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			paths = append(paths, path)
		}
		return nil
	})
	if err != nil {
		archive.Close()
		return err
	}
	sort.Strings(paths)
	for _, path := range paths {
		relative, _ := filepath.Rel(source, path)
		info, err := os.Stat(path)
		if err != nil {
			archive.Close()
			return err
		}
		header := &zip.FileHeader{Name: "select/" + filepath.ToSlash(relative), Method: zip.Deflate}
		header.SetMode(info.Mode())
		entry, err := archive.CreateHeader(header)
		if err != nil {
			archive.Close()
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			archive.Close()
			return err
		}
		if _, err := entry.Write(data); err != nil {
			archive.Close()
			return err
		}
	}
	if err := archive.Close(); err != nil {
		return err
	}
	return file.Close()
}

func atomicWrite(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".release-*")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		file.Close()
		return err
	}
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}

func replacePackage(source, destination string) error {
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0755); err != nil {
		return err
	}
	// Stage on the destination filesystem so directory renames remain atomic.
	work, err := os.MkdirTemp(parent, ".release-select-*")
	if err != nil {
		return err
	}
	removeWork := true
	defer func() {
		if removeWork {
			os.RemoveAll(work)
		}
	}()
	stage, previous := filepath.Join(work, "new"), filepath.Join(work, "previous")
	if err := copyPath(source, stage); err != nil {
		return err
	}
	// copyPath normalizes permissions; restore the executable files explicitly.
	if err := filepath.WalkDir(filepath.Join(stage, "server"), func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		return os.Chmod(path, 0755)
	}); err != nil {
		return err
	}
	hadPrevious := false
	if info, err := os.Lstat(destination); err == nil {
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return errors.New("published/select must be a real directory")
		}
		if err := os.Rename(destination, previous); err != nil {
			return err
		}
		hadPrevious = true
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(stage, destination); err != nil {
		if hadPrevious {
			if restore := os.Rename(previous, destination); restore != nil {
				// Retain the backup if recovery needs manual intervention.
				removeWork = false
				return fmt.Errorf("publish failed (%v); restore failed (%v); previous package: %s", err, restore, previous)
			}
		}
		return err
	}
	return nil
}
