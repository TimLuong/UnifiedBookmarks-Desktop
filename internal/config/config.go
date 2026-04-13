package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Config struct {
	OpenAIKey    string
	OpenAIBase   string
	OpenAIModel  string
	BatchSize    int
	MaxRetries   int
	BackupDir    string
	SystemPrompt string // custom prompt override, loaded from ai-prompt.txt
}

func Load() *Config {
	exe, _ := os.Executable()
	base := filepath.Dir(exe)

	// Also try loading .env from the executable directory
	loadEnvFile(filepath.Join(base, ".env"))
	// And from current working directory
	if cwd, err := os.Getwd(); err == nil {
		loadEnvFile(filepath.Join(cwd, ".env"))
	}

	batchSize, _ := strconv.Atoi(envOr("BATCH_SIZE", "0")) // 0 = unlimited (one batch)
	maxRetries, _ := strconv.Atoi(envOr("MAX_RETRIES", "3"))

	backupDir := envOr("BACKUP_DIR", filepath.Join(base, "backups"))
	if !filepath.IsAbs(backupDir) {
		backupDir = filepath.Join(base, backupDir)
	}

	// Load custom AI prompt from ai-prompt.txt (supports multiline)
	var customPrompt string
	if data, err := os.ReadFile(filepath.Join(base, "ai-prompt.txt")); err == nil {
		customPrompt = strings.TrimSpace(string(data))
	}

	return &Config{
		OpenAIKey:    os.Getenv("OPENAI_API_KEY"),
		OpenAIBase:   os.Getenv("OPENAI_BASE_URL"),
		OpenAIModel:  envOr("OPENAI_MODEL", "gpt-4o-mini"),
		BatchSize:    batchSize,
		MaxRetries:   maxRetries,
		BackupDir:    backupDir,
		SystemPrompt: customPrompt,
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func loadEnvFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	lines := splitLines(string(data))
	for _, line := range lines {
		if len(line) == 0 || line[0] == '#' {
			continue
		}
		for i := 0; i < len(line); i++ {
			if line[i] == '=' {
				key := trimSpace(line[:i])
				val := trimSpace(line[i+1:])
				// Strip inline comments
				for j := 0; j < len(val); j++ {
					if val[j] == '#' && j > 0 && val[j-1] == ' ' {
						val = trimSpace(val[:j])
						break
					}
				}
				if os.Getenv(key) == "" {
					os.Setenv(key, val)
				}
				break
			}
		}
	}
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			line := s[start:i]
			if len(line) > 0 && line[len(line)-1] == '\r' {
				line = line[:len(line)-1]
			}
			lines = append(lines, line)
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func trimSpace(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}
