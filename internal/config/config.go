package config

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// LogSource represents a log source with its path and label
type LogSource struct {
	Path  string
	Label string
	Panel int
}

// Config holds the global LogPulse configuration
type Config struct {
	Sources     []LogSource
	RefreshRate int
	MaxLines    int
	FilePath    string // path to the active .conf file
}

// confFileName is the configuration file name
const confFileName = "logpulse.conf"

// defaultSources are the default paths (used only if no .conf file exists)
var defaultSources = []LogSource{
	{Path: "", Label: "System", Panel: 0},
	{Path: "", Label: "Application", Panel: 1},
	{Path: "", Label: "Web", Panel: 2},
	{Path: "", Label: "Database", Panel: 3},
}

// DefaultConfig returns a configuration with default values
func DefaultConfig() *Config {
	return &Config{
		RefreshRate: 200,
		MaxLines:    1000,
		Sources:     append([]LogSource{}, defaultSources...),
		FilePath:    confPath(),
	}
}

// confPath returns the path to the configuration file:
// same directory as the executable, or the working directory.
func confPath() string {
	exe, err := os.Executable()
	if err != nil {
		return confFileName
	}
	return filepath.Join(filepath.Dir(exe), confFileName)
}

// Load loads the configuration: tries to read the .conf first;
// if not found, writes the defaults and uses them.
func Load() *Config {
	cfg := DefaultConfig()

	// Allow overriding the .conf path via flag
	confFile := flag.String("conf", cfg.FilePath, "Path to the configuration file")
	flag.Parse()
	cfg.FilePath = *confFile

	if err := cfg.ReadFile(); err != nil {
		// File not found or unreadable → save defaults
		_ = cfg.Save()
	}
	return cfg
}

// ReadFile reads and parses the .conf file.
func (c *Config) ReadFile() error {
	f, err := os.Open(c.FilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "sys":
			c.Sources[0].Path = val
		case "app":
			c.Sources[1].Path = val
		case "web":
			c.Sources[2].Path = val
		case "db":
			c.Sources[3].Path = val
		}
	}
	return scanner.Err()
}

// Save writes the current configuration to the .conf file
func (c *Config) Save() error {
	f, err := os.Create(c.FilePath)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	_, err = fmt.Fprintf(f,
		"# LogPulse - Log file path configuration\n"+
			"# Leave a value empty to disable that panel\n\n"+
			"sys = %s\n"+
			"app = %s\n"+
			"web = %s\n"+
			"db  = %s\n",
		c.Sources[0].Path,
		c.Sources[1].Path,
		c.Sources[2].Path,
		c.Sources[3].Path,
	)
	return err
}

// FileExists reports whether the given file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}
