// Native developer tooling. Packaging is explicit and never runs during build/check.
package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

var targets = map[string]bool{"darwin/amd64": true, "darwin/arm64": true, "windows/amd64": true, "windows/arm64": true, "linux/amd64": true, "linux/arm64": true}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func readJSON(p string) (map[string]any, error) {
	b, e := os.ReadFile(p)
	if e != nil {
		return nil, e
	}
	var v map[string]any
	e = json.Unmarshal(b, &v)
	return v, e
}
func jsonBytes(v any) []byte {
	b, e := json.MarshalIndent(v, "", "  ")
	if e != nil {
		panic(e)
	}
	return append(b, '\n')
}
func run() error {
	if len(os.Args) < 2 {
		return errors.New("usage: go run ./scripts <build|dev|check|package|notices> [-target os/arch] [-binary path]")
	}
	command := os.Args[1]
	f := flag.NewFlagSet(command, flag.ContinueOnError)
	target := f.String("target", runtime.GOOS+"/"+runtime.GOARCH, "target OS/Go architecture")
	binary := f.String("binary", "", "existing executable for packaging")
	headless := f.Bool("headless", false, "development-only core build without GUI")
	if err := f.Parse(os.Args[2:]); err != nil {
		return err
	}
	if f.NArg() != 0 {
		return errors.New("unexpected arguments")
	}
	if !targets[*target] {
		return errors.New("supported targets: darwin/windows/linux with amd64 or arm64")
	}
	root, e := os.Getwd()
	if e != nil {
		return e
	}
	meta, e := readJSON(filepath.Join(root, "plugin.json"))
	if e != nil {
		return errors.New("run this command from the project folder")
	}
	version, _ := meta["version"].(string)
	parts := strings.Split(*target, "/")
	name := "select"
	if parts[0] == "windows" {
		name += ".exe"
	}
	buildPath := filepath.Join(root, "build", strings.ReplaceAll(*target, "/", "-"), name)
	switch command {
	case "build", "dev":
		if command == "dev" && (*headless || *target != runtime.GOOS+"/"+runtime.GOARCH) {
			return errors.New("the development plugin needs a GUI build for this machine")
		}
		if e = check(root); e != nil {
			return e
		}
		if parts[0] != "windows" && runtime.GOOS != parts[0] && !*headless {
			return errors.New("GUI builds for macOS/Linux require the matching host and native SDKs; use the native OS or -headless for compile checks")
		}
		if parts[0] == "linux" && !*headless {
			if e = exec.Command("pkg-config", "--exists", "gtk+-3.0", "webkit2gtk-4.1").Run(); e != nil {
				return errors.New("Linux GUI builds need GCC, pkg-config, GTK3 and WebKitGTK 4.1 development packages")
			}
		}
		if e = os.MkdirAll(filepath.Dir(buildPath), 0755); e != nil {
			return e
		}
		tags := "production"
		if parts[0] == "linux" {
			tags += ",webkit2_41"
		}
		if *headless {
			tags = "headless"
			buildPath = filepath.Join(filepath.Dir(buildPath), "core-"+name)
		}
		link := "-s -w"
		if parts[0] == "windows" && !*headless {
			link += " -H windowsgui"
		}
		cmd := exec.Command("go", "build", "-trimpath", "-tags", tags, "-ldflags", link, "-o", buildPath, "./src")
		cmd.Env = append(os.Environ(), "GOOS="+parts[0], "GOARCH="+parts[1])
		if parts[0] == "windows" || (*headless && parts[0] != "darwin") {
			cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
		} else {
			cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if e = cmd.Run(); e != nil {
			return e
		}
		fmt.Println(buildPath)
		if command == "dev" {
			return development(root, buildPath)
		}
		return nil
	case "check":
		return check(root)
	case "notices":
		return notices(root)
	case "package":
		if *headless {
			return errors.New("headless development builds cannot be packaged")
		}
		if e = check(root); e != nil {
			return e
		}
		p := *binary
		if p == "" {
			p = buildPath
		}
		if e = verifyBinary(p, parts[0], parts[1]); e != nil {
			return e
		}
		return pack(root, p, name, *target, version)
	default:
		return errors.New("unknown command; use build, dev, check, notices or package")
	}
}
func check(root string) error {
	var version string
	for _, name := range []string{"plugin.json", ".claude-plugin/plugin.json", ".codex-plugin/plugin.json"} {
		v, e := readJSON(filepath.Join(root, name))
		if e != nil {
			return e
		}
		if v["name"] != "select" {
			return errors.New("plugin identity must remain select")
		}
		s, _ := v["version"].(string)
		if version != "" && version != s {
			return errors.New("manifest versions differ")
		}
		version = s
	}
	for _, name := range []string{"bin", "node_modules", "package.json", "package-lock.json"} {
		if _, e := os.Stat(filepath.Join(root, name)); e == nil {
			return fmt.Errorf("obsolete Node runtime path remains: %s", name)
		}
	}
	for _, name := range []string{"assets/icon.png", "src/main.go", "src/ui/index.html", "src/ui/app.js", "skills/db-setup/SKILL.md", "skills/database-query/SKILL.md", "go.mod", "go.sum"} {
		if _, e := os.Stat(filepath.Join(root, name)); e != nil {
			return e
		}
	}
	for _, p := range []struct{ name, token string }{{"mcp.json", "${PLUGIN_ROOT}"}, {".mcp.json", "${CLAUDE_PLUGIN_ROOT}"}} {
		v, e := readJSON(filepath.Join(root, p.name))
		if e != nil {
			return e
		}
		servers, ok := v["mcpServers"].(map[string]any)
		if !ok || len(servers) != 1 {
			return errors.New("one select MCP server is required")
		}
		s, ok := servers["select"].(map[string]any)
		if !ok {
			return errors.New("select server is missing")
		}
		if s["command"] != p.token+"/server/select" {
			return errors.New("source manifests must launch the native executable directly")
		}
		a, ok := s["args"].([]any)
		if !ok || len(a) != 1 || a[0] != "mcp" {
			return errors.New("native server must use mcp mode")
		}
	}
	fmt.Println("Native source layout and plugin manifests validated.")
	return nil
}
func verifyBinary(p, osName, arch string) error {
	// Use Go build metadata to prevent packaging the wrong target or core-only binary.
	cmd := exec.Command("go", "version", "-m", p)
	b, e := cmd.Output()
	if e != nil {
		return errors.New("build the native executable first")
	}
	s := string(b)
	for _, part := range []string{"GOOS=" + osName, "GOARCH=" + arch} {
		if !strings.Contains(s, part) {
			return fmt.Errorf("executable target does not match %s/%s", osName, arch)
		}
	}
	if strings.Contains(s, "headless") {
		return errors.New("cannot package a headless development executable")
	}
	return nil
}
func notices(root string) error {
	cmd := exec.Command("go", "list", "-m", "-json", "all")
	b, e := cmd.Output()
	if e != nil {
		return e
	}
	d := json.NewDecoder(bytes.NewReader(b))
	var out strings.Builder
	out.WriteString("Select native Go dependencies. Versions are pinned in go.mod/go.sum.\n\n")
	for {
		var m struct {
			Path, Version, Dir string
			Main               bool
		}
		if e = d.Decode(&m); e == io.EOF {
			break
		} else if e != nil {
			return e
		}
		if m.Main {
			continue
		}
		fmt.Fprintf(&out, "\n===== %s %s =====\n", m.Path, m.Version)
		files, _ := os.ReadDir(m.Dir)
		for _, f := range files {
			n := strings.ToLower(f.Name())
			if !f.IsDir() && (strings.HasPrefix(n, "license") || strings.HasPrefix(n, "copying") || n == "notice") {
				data, err := os.ReadFile(filepath.Join(m.Dir, f.Name()))
				if err == nil {
					fmt.Fprintf(&out, "\n%s\n%s\n", f.Name(), data)
				}
			}
		}
	}
	lines := strings.Split(out.String(), "\n")
	for i := range lines {
		lines[i] = strings.TrimRight(lines[i], " \t\r")
	}
	return os.WriteFile(filepath.Join(root, "THIRD_PARTY_NOTICES.txt"), []byte(strings.Join(lines, "\n")), 0644)
}

type archiveFile struct {
	data []byte
	mode os.FileMode
}

// Install this wrapper once. Its absolute command stays outside the host's
// installed plugin cache, so subsequent builds replace the executable it uses.
func development(root, binary string) error {
	files := map[string]archiveFile{}
	for _, name := range []string{"plugin.json", ".claude-plugin", ".codex-plugin", "mcp.json", ".mcp.json", "skills", "assets", "README.md", "docs/architecture.md", "docs/maintenance.md", "docs/extension-setup.md", "docs/team-distribution.md", "docs/references.md"} {
		if err := addTree(files, root, name); err != nil {
			return err
		}
	}
	for _, name := range []string{"mcp.json", ".mcp.json"} {
		config, err := readJSON(filepath.Join(root, name))
		if err != nil {
			return err
		}
		server := config["mcpServers"].(map[string]any)["select"].(map[string]any)
		server["command"] = binary
		files[name] = archiveFile{jsonBytes(config), 0644}
	}
	files["DEVELOPMENT.txt"] = archiveFile{[]byte("Local development wrapper for this machine only.\nExecutable: " + binary + "\n\nInstall Select once (plugin identifier: select). When switching from db-plugin, add it as a new plugin and disable the old plugin to avoid duplicate tools.\nCSS edits apply live from src/ui. Reopen setup for HTML/JavaScript edits.\nFor Go changes, rebuild with: go run ./scripts build\nClose the existing Select settings window and restart the MCP session/app after rebuilding.\nCode and UI changes do not need another upload.\nSkills, manifests or repository location changes require updating this wrapper.\nDo not distribute this machine-specific archive.\n"), 0644}
	dir := filepath.Join(root, "build", "dev")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	archive := filepath.Join(dir, "select-dev.zip")
	if err := writeZIP(archive, "select/", files); err != nil {
		return err
	}
	fmt.Println(archive)
	fmt.Println("Install this development wrapper once in each desktop host. Future code/UI changes need a rebuild and a new MCP session, not another ZIP upload.")
	return nil
}

func addTree(files map[string]archiveFile, root, relative string) error {
	return filepath.WalkDir(filepath.Join(root, relative), func(p string, d os.DirEntry, e error) error {
		if e != nil {
			return e
		}
		if d.IsDir() {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return errors.New("symlinks cannot be packaged")
		}
		b, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		rel, _ := filepath.Rel(root, p)
		files[filepath.ToSlash(rel)] = archiveFile{b, 0644}
		return nil
	})
}
func writeZIP(p, prefix string, files map[string]archiveFile) error {
	f, e := os.Create(p)
	if e != nil {
		return e
	}
	w := zip.NewWriter(f)
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, name := range names {
		if strings.HasPrefix(name, "bin/") || strings.Contains(name, "..") || strings.HasSuffix(name, ".zip") {
			w.Close()
			f.Close()
			return errors.New("invalid archive path")
		}
		item := files[name]
		h := &zip.FileHeader{Name: prefix + name, Method: zip.Deflate}
		h.SetMode(item.mode)
		entry, e := w.CreateHeader(h)
		if e != nil {
			w.Close()
			f.Close()
			return e
		}
		if _, e = entry.Write(item.data); e != nil {
			w.Close()
			f.Close()
			return e
		}
	}
	e = w.Close()
	if closeErr := f.Close(); e == nil {
		e = closeErr
	}
	return e
}
func pack(root, binary, name, target, version string) error {
	files := map[string]archiveFile{}
	for _, p := range []string{"plugin.json", ".claude-plugin", ".codex-plugin", "mcp.json", ".mcp.json", "skills", "assets", "README.md", "docs/architecture.md", "docs/maintenance.md", "docs/extension-setup.md", "docs/team-distribution.md", "docs/references.md", "THIRD_PARTY_NOTICES.txt"} {
		if e := addTree(files, root, p); e != nil {
			return e
		}
	}
	for _, p := range []struct{ name, token string }{{"mcp.json", "${PLUGIN_ROOT}"}, {".mcp.json", "${CLAUDE_PLUGIN_ROOT}"}} {
		v, _ := readJSON(filepath.Join(root, p.name))
		server := v["mcpServers"].(map[string]any)["select"].(map[string]any)
		server["command"] = p.token + "/server/" + name
		files[p.name] = archiveFile{jsonBytes(v), 0644}
	}
	b, e := os.ReadFile(binary)
	if e != nil {
		return e
	}
	files["server/"+name] = archiveFile{b, 0755}
	dir := filepath.Join(root, "releases")
	if e = os.MkdirAll(dir, 0755); e != nil {
		return e
	}
	// WIP packages use stable filenames; versions remain in required metadata.
	suffix := strings.ReplaceAll(target, "/", "-")
	paths := []string{}
	pluginPath := filepath.Join(dir, "select-"+suffix+".zip")
	if e = writeZIP(pluginPath, "select/", files); e != nil {
		return e
	}
	paths = append(paths, pluginPath)
	market := map[string]archiveFile{}
	for p, v := range files {
		market["plugins/select/"+p] = v
	}
	market[".claude-plugin/marketplace.json"] = archiveFile{jsonBytes(map[string]any{"name": "select-team", "owner": map[string]string{"name": "Select contributors"}, "plugins": []any{map[string]any{"name": "select", "source": "./plugins/select", "version": version}}}), 0644}
	market[".agents/plugins/marketplace.json"] = archiveFile{jsonBytes(map[string]any{"name": "select-team", "plugins": []any{map[string]any{"name": "select", "source": map[string]string{"source": "local", "path": "./plugins/select"}, "policy": map[string]string{"installation": "AVAILABLE", "authentication": "ON_INSTALL"}, "category": "Developer Tools"}}}), 0644}
	marketPath := filepath.Join(dir, "select-marketplace-"+suffix+".zip")
	if e = writeZIP(marketPath, "select-marketplace/", market); e != nil {
		return e
	}
	paths = append(paths, marketPath)
	extension, e := readJSON(filepath.Join(root, "docs/desktop-extension.json"))
	if e != nil {
		return e
	}
	extension["version"] = version
	extension["compatibility"] = map[string]any{"platforms": []string{strings.Split(target, "/")[0]}}
	extension["server"] = map[string]any{"type": "binary", "entry_point": "server/" + name, "mcp_config": map[string]any{"command": "${__dirname}/server/" + name, "args": []string{"desktop"}, "env": map[string]string{"DB_MCPB_PROJECT": "${user_config.project}", "DB_MCPB_ENVIRONMENT": "${user_config.environment}", "DB_MCPB_HOST": "${user_config.host}", "DB_MCPB_PORT": "${user_config.port}", "DB_MCPB_DATABASE": "${user_config.database}", "DB_MCPB_USER": "${user_config.user}", "DB_MCPB_PASSWORD": "${user_config.password}", "DB_MCPB_TLS": "${user_config.tls}", "DB_MCPB_CA_FILE": "${user_config.ca_file}"}}}
	extFiles := map[string]archiveFile{"manifest.json": {jsonBytes(extension), 0644}, "server/" + name: {b, 0755}, "THIRD_PARTY_NOTICES.txt": files["THIRD_PARTY_NOTICES.txt"], "assets/icon.png": files["assets/icon.png"]}
	extPath := filepath.Join(dir, "select-connection-"+suffix+".mcpb")
	if e = writeZIP(extPath, "", extFiles); e != nil {
		return e
	}
	paths = append(paths, extPath)
	var sums strings.Builder
	for _, p := range paths {
		data, e := os.ReadFile(p)
		if e != nil {
			return e
		}
		sum := sha256.Sum256(data)
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), filepath.Base(p))
		fmt.Println(p)
	}
	return os.WriteFile(filepath.Join(dir, "SHA256SUMS-"+suffix+".txt"), []byte(sums.String()), 0644)
}
