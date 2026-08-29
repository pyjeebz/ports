package inspect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectProject(t *testing.T) {
	tmpDir := t.TempDir()

	// Test 1: .git wins (outermost)
	gitRoot := filepath.Join(tmpDir, "monorepo")
	gitSub := filepath.Join(gitRoot, "apps", "frontend")
	_ = os.MkdirAll(gitSub, 0o755)
	_ = os.WriteFile(filepath.Join(gitRoot, ".git"), []byte{}, 0o644)
	_ = os.WriteFile(filepath.Join(gitSub, "package.json"), []byte{}, 0o644)

	if got := DetectProject(gitSub); got != gitRoot {
		t.Errorf("Test 1: DetectProject(%q) = %q, want %q", gitSub, got, gitRoot)
	}

	// Test 2: turbo.json (Turborepo)
	turboRoot := filepath.Join(tmpDir, "turborepo")
	turboSub := filepath.Join(turboRoot, "apps", "web")
	_ = os.MkdirAll(turboSub, 0o755)
	_ = os.WriteFile(filepath.Join(turboRoot, "turbo.json"), []byte(`{"pipeline": {}}`), 0o644)
	_ = os.WriteFile(filepath.Join(turboSub, "package.json"), []byte{}, 0o644)

	if got := DetectProject(turboSub); got != turboRoot {
		t.Errorf("Test 2: DetectProject(%q) = %q, want %q", turboSub, got, turboRoot)
	}

	// Test 3: nx.json (Nx)
	nxRoot := filepath.Join(tmpDir, "nxrepo")
	nxSub := filepath.Join(nxRoot, "packages", "ui")
	_ = os.MkdirAll(nxSub, 0o755)
	_ = os.WriteFile(filepath.Join(nxRoot, "nx.json"), []byte(`{"targets": {}}`), 0o644)
	_ = os.WriteFile(filepath.Join(nxSub, "package.json"), []byte{}, 0o644)

	if got := DetectProject(nxSub); got != nxRoot {
		t.Errorf("Test 3: DetectProject(%q) = %q, want %q", nxSub, got, nxRoot)
	}

	// Test 4: pnpm-workspace.yaml
	pnpmRoot := filepath.Join(tmpDir, "pnpmrepo")
	pnpmSub := filepath.Join(pnpmRoot, "packages", "lib")
	_ = os.MkdirAll(pnpmSub, 0o755)
	_ = os.WriteFile(filepath.Join(pnpmRoot, "pnpm-workspace.yaml"), []byte(`packages:\n  - packages/*`), 0o644)
	_ = os.WriteFile(filepath.Join(pnpmSub, "package.json"), []byte{}, 0o644)

	if got := DetectProject(pnpmSub); got != pnpmRoot {
		t.Errorf("Test 4: DetectProject(%q) = %q, want %q", pnpmSub, got, pnpmRoot)
	}

	// Test 5: yarn.lock with workspaces
	yarnRoot := filepath.Join(tmpDir, "yarnrepo")
	yarnSub := filepath.Join(yarnRoot, "packages", "shared")
	_ = os.MkdirAll(yarnSub, 0o755)
	_ = os.WriteFile(filepath.Join(yarnRoot, "yarn.lock"), []byte(`workspaces:\n  - packages/*`), 0o644)
	_ = os.WriteFile(filepath.Join(yarnSub, "package.json"), []byte{}, 0o644)

	if got := DetectProject(yarnSub); got != yarnRoot {
		t.Errorf("Test 5: DetectProject(%q) = %q, want %q", yarnSub, got, yarnRoot)
	}

	// Test 6: go.work
	goRoot := filepath.Join(tmpDir, "gowork")
	goSub := filepath.Join(goRoot, "services", "api")
	_ = os.MkdirAll(goSub, 0o755)
	_ = os.WriteFile(filepath.Join(goRoot, "go.work"), []byte(`go 1.21\n\nuse (\n  ./services/api\n)`), 0o644)
	_ = os.WriteFile(filepath.Join(goSub, "go.mod"), []byte(`module api\n\ngo 1.21`), 0o644)

	if got := DetectProject(goSub); got != goRoot {
		t.Errorf("Test 6: DetectProject(%q) = %q, want %q", goSub, got, goRoot)
	}

	// Test 7: Cargo.workspace
	cargoRoot := filepath.Join(tmpDir, "cargowork")
	cargoSub := filepath.Join(cargoRoot, "crates", "core")
	_ = os.MkdirAll(cargoSub, 0o755)
	_ = os.WriteFile(filepath.Join(cargoRoot, "Cargo.toml"), []byte(`[workspace]\nmembers = ["crates/*"]`), 0o644)
	_ = os.WriteFile(filepath.Join(cargoSub, "Cargo.toml"), []byte(`[package]\nname = "core"`), 0o644)

	if got := DetectProject(cargoSub); got != cargoRoot {
		t.Errorf("Test 7: DetectProject(%q) = %q, want %q", cargoSub, got, cargoRoot)
	}

	// Test 8: home directory is never a project
	home, _ := os.UserHomeDir()
	if got := DetectProject(home); got != "" {
		t.Errorf("Test 8: DetectProject(home) = %q, want empty", got)
	}

	// Test 9: empty cwd returns empty
	if got := DetectProject(""); got != "" {
		t.Errorf("Test 9: DetectProject(empty) = %q, want empty", got)
	}
}
