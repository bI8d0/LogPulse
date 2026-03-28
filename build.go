// build.go — cross-compiles LogPulse for Linux from any OS.
// Run with: go run build.go

//go:build ignore

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

func main() {
	_, currentFile, _, _ := runtime.Caller(0)
	projectDir, err := filepath.Abs(filepath.Dir(currentFile))
	if err != nil {
		fatal("Could not determine project directory: %v", err)
	}

	outputName := "logpulse"

	fmt.Println("╔══════════════════════════════════╗")
	fmt.Println("║   LogPulse — Linux Build         ║")
	fmt.Println("╚══════════════════════════════════╝")
	fmt.Printf("  Directory : %s\n", projectDir)
	fmt.Printf("  Target    : linux/amd64\n")
	fmt.Printf("  Output    : %s\n\n", outputName)

	fmt.Println("▶ Downloading dependencies (go mod tidy)...")
	run(projectDir, "go", "mod", "tidy")

	fmt.Println("▶ Compiling for linux/amd64...")
	cmd := exec.Command("go", "build", "-ldflags=-s -w", "-o", outputName, ".")
	cmd.Dir = projectDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(),
		"GOOS=linux",
		"GOARCH=amd64",
		"CGO_ENABLED=0",
	)

	if err := cmd.Run(); err != nil {
		fatal("Compilation failed: %v", err)
	}

	outPath := filepath.Join(projectDir, outputName)
	info, err := os.Stat(outPath)
	size := int64(0)
	if err == nil {
		size = info.Size()
	}

	fmt.Printf("\n✔ Binary generated: %s (%.1f KB)\n", outPath, float64(size)/1024)
	fmt.Println("  Copy it to your Linux machine and run:")
	fmt.Printf("  chmod +x %s && ./%s\n", outputName, outputName)
}

func run(dir string, name string, args ...string) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fatal("Error running '%s %v': %v", name, args, err)
	}
}

func fatal(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "✗ "+format+"\n", args...)
	os.Exit(1)
}
