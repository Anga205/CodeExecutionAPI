package executor

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

var syscallMount = syscall.Mount
var syscallUnmount = syscall.Unmount

// SandboxWorkspace describes the on-disk layout and launch plan for a single
// sandboxed execution. It is exported so the integration tests in the tests
// package can assert on the isolation plan without duplicating production logic.
type SandboxWorkspace struct {
	Dir                 string
	SourcePath          string
	BinaryPath          string
	SandboxInitHostPath string
	SandboxInitExecPath string
	CompileCommand      []string
	RunCommand          []string
	ExtraEnv            []string
	UseChroot           bool
	RuntimeMounts       []string
	MountedTargets      []string
}

func prepareWorkspace(language, code string) (SandboxWorkspace, error) {
	dir, err := os.MkdirTemp("", "codesandbox-*")
	if err != nil {
		return SandboxWorkspace{}, fmt.Errorf("failed to create sandbox dir: %w", err)
	}
	if err := os.Chown(dir, 65534, 65534); err != nil {
		_ = os.RemoveAll(dir)
		return SandboxWorkspace{}, fmt.Errorf("failed to set sandbox dir ownership: %w", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		_ = os.RemoveAll(dir)
		return SandboxWorkspace{}, fmt.Errorf("failed to set sandbox dir permissions: %w", err)
	}
	ws, err := buildWorkspacePlan(dir, language)
	if err != nil {
		_ = os.RemoveAll(dir)
		return SandboxWorkspace{}, err
	}
	if err := os.WriteFile(ws.SourcePath, []byte(code), 0o644); err != nil {
		_ = os.RemoveAll(dir)
		return SandboxWorkspace{}, fmt.Errorf("failed to write source: %w", err)
	}
	return ws, nil
}

func buildWorkspacePlan(dir, language string) (SandboxWorkspace, error) {
	normalized := normalizeLanguage(language)
	ws := SandboxWorkspace{
		Dir:                 dir,
		BinaryPath:          filepath.Join(dir, "program"),
		SandboxInitHostPath: filepath.Join(dir, "sandbox-init"),
		SandboxInitExecPath: "/sandbox-init",
	}

	switch normalized {
	case "c":
		ws.SourcePath = filepath.Join(dir, "main.c")
		ws.CompileCommand = []string{"gcc", "-O2", "-pipe", "-std=c11", "-static", "-s", "-o", ws.BinaryPath, ws.SourcePath}
		ws.RunCommand = []string{"/program"}
		ws.UseChroot = true
	case "cpp":
		ws.SourcePath = filepath.Join(dir, "main.cpp")
		ws.CompileCommand = []string{"g++", "-O2", "-pipe", "-std=c++17", "-static", "-s", "-o", ws.BinaryPath, ws.SourcePath}
		ws.RunCommand = []string{"/program"}
		ws.UseChroot = true
	case "java":
		javaBin, javaLibDir, javaHome, err := resolveJavaRuntime()
		if err != nil {
			return SandboxWorkspace{}, err
		}
		ws.SourcePath = filepath.Join(dir, "Main.java")
		ws.CompileCommand = []string{"javac", ws.SourcePath}
		ws.RunCommand = []string{javaBin, "-Xms8m", "-Xmx24m", "-cp", "/", "Main"}
		ws.ExtraEnv = []string{"LD_LIBRARY_PATH=" + javaLibDir, "JAVA_HOME=" + javaHome}
		ws.UseChroot = true
		ws.RuntimeMounts = defaultRuntimeMounts()
		securityFile := filepath.Join(javaHome, "conf", "security", "java.security")
		if resolvedSecurityFile, resolveErr := filepath.EvalSymlinks(securityFile); resolveErr == nil {
			ws.RuntimeMounts = append(ws.RuntimeMounts, filepath.Dir(resolvedSecurityFile))
		}
	case "python3":
		ws.SourcePath = filepath.Join(dir, "main.py")
		ws.RunCommand = []string{"/usr/bin/python3", "/main.py"}
		ws.UseChroot = true
		ws.RuntimeMounts = defaultRuntimeMounts()
	default:
		return SandboxWorkspace{}, fmt.Errorf("unsupported language: %s", language)
	}

	return ws, nil
}

func normalizeLanguage(language string) string {
	switch strings.ToLower(strings.TrimSpace(language)) {
	case "python", "py", "python3":
		return "python3"
	case "c++", "cpp":
		return "cpp"
	default:
		return strings.ToLower(strings.TrimSpace(language))
	}
}

func resolveJavaRuntime() (javaBin string, javaLibDir string, javaHome string, err error) {
	javaPath, lookErr := exec.LookPath("java")
	if lookErr != nil {
		return "", "", "", fmt.Errorf("java runtime not found: %w", lookErr)
	}

	resolvedJava, evalErr := filepath.EvalSymlinks(javaPath)
	if evalErr != nil {
		resolvedJava = javaPath
	}

	javaBin = resolvedJava
	javaHome = filepath.Clean(filepath.Join(filepath.Dir(resolvedJava), ".."))

	libDirs := []string{
		filepath.Join(javaHome, "lib"),
		filepath.Join(javaHome, "lib", "jli"),
		filepath.Join(javaHome, "lib", "server"),
		filepath.Join(javaHome, "jre", "lib"),
		filepath.Join(javaHome, "jre", "lib", "jli"),
		filepath.Join(javaHome, "jre", "lib", "server"),
	}
	if archJli, globErr := filepath.Glob(filepath.Join(javaHome, "lib", "*", "jli")); globErr == nil {
		libDirs = append(libDirs, archJli...)
	}
	if archServer, globErr := filepath.Glob(filepath.Join(javaHome, "lib", "*", "server")); globErr == nil {
		libDirs = append(libDirs, archServer...)
	}
	if jreArchJli, globErr := filepath.Glob(filepath.Join(javaHome, "jre", "lib", "*", "jli")); globErr == nil {
		libDirs = append(libDirs, jreArchJli...)
	}
	if jreArchServer, globErr := filepath.Glob(filepath.Join(javaHome, "jre", "lib", "*", "server")); globErr == nil {
		libDirs = append(libDirs, jreArchServer...)
	}
	var existingLibDirs []string
	for _, dir := range libDirs {
		info, statErr := os.Stat(dir)
		if statErr == nil && info.IsDir() {
			existingLibDirs = append(existingLibDirs, dir)
		}
	}
	if len(existingLibDirs) == 0 {
		existingLibDirs = []string{filepath.Join(javaHome, "lib")}
	}
	javaLibDir = strings.Join(existingLibDirs, ":")
	return javaBin, javaLibDir, javaHome, nil
}

func defaultRuntimeMounts() []string {
	return []string{"/usr", "/lib", "/lib64", "/bin"}
}

func prepareRuntimeMounts(ws *SandboxWorkspace) error {
	if !ws.UseChroot || len(ws.RuntimeMounts) == 0 {
		return nil
	}

	if err := syscallMount("", "/", "", syscall.MS_REC|syscall.MS_PRIVATE, ""); err != nil {
		cleanupRuntimeMounts(ws)
		return fmt.Errorf("failed to mark root mount propagation private: %w", err)
	}

	for _, source := range ws.RuntimeMounts {
		sourceInfo, err := os.Lstat(source)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to inspect runtime mount source %s: %w", source, err)
		}

		target := filepath.Join(ws.Dir, strings.TrimPrefix(source, "/"))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to prepare mount target %s: %w", target, err)
		}

		if sourceInfo.Mode()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(source)
			if err != nil {
				cleanupRuntimeMounts(ws)
				return fmt.Errorf("failed to read runtime mount symlink %s: %w", source, err)
			}
			if err := os.RemoveAll(target); err != nil {
				cleanupRuntimeMounts(ws)
				return fmt.Errorf("failed to reset symlink target %s: %w", target, err)
			}
			if err := os.Symlink(linkTarget, target); err != nil {
				cleanupRuntimeMounts(ws)
				return fmt.Errorf("failed to mirror symlink %s -> %s: %w", target, linkTarget, err)
			}
			continue
		}

		if err := os.MkdirAll(target, 0o755); err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to prepare mount target %s: %w", target, err)
		}
		if err := bindMountReadOnly(source, target); err != nil {
			cleanupRuntimeMounts(ws)
			return fmt.Errorf("failed to prepare read-only bind mount %s -> %s: %w", source, target, err)
		}
		ws.MountedTargets = append(ws.MountedTargets, target)
	}

	return nil
}

func bindMountReadOnly(source, target string) error {
	if err := syscallMount(source, target, "", syscall.MS_BIND|syscall.MS_REC, ""); err != nil {
		return err
	}
	if err := syscallMount("", target, "", syscall.MS_BIND|syscall.MS_REMOUNT|syscall.MS_RDONLY|syscall.MS_NOSUID|syscall.MS_NODEV|syscall.MS_REC, ""); err != nil {
		_ = syscallUnmount(target, syscall.MNT_DETACH)
		return err
	}
	return nil
}

func cleanupRuntimeMounts(ws *SandboxWorkspace) {
	for i := len(ws.MountedTargets) - 1; i >= 0; i-- {
		target := ws.MountedTargets[i]
		_ = syscall.Unmount(target, syscall.MNT_DETACH)
	}
	ws.MountedTargets = nil
}

func compileWorkspace(ws SandboxWorkspace) (string, bool) {
	if len(ws.CompileCommand) == 0 {
		return "", true
	}

	cmd := exec.Command(ws.CompileCommand[0], ws.CompileCommand[1:]...)
	cmd.Dir = ws.Dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), false
	}
	return "", true
}
