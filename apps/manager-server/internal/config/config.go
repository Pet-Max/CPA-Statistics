package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const configEnvKey = "CPA_MANAGER_CONFIG"

const defaultConfigName = "config.yaml"
const legacyDefaultConfigName = "config.json"

const defaultSecretFile = "/run/secrets/cpa_management_key"
const defaultAdminSecretFile = "/run/secrets/cpa_admin_key"
const defaultDataKeySecretFile = "/run/secrets/cpa_data_key"

type Config struct {
	HTTPAddr       string
	DataDir        string
	DBPath         string
	CPAUpstreamURL string
	ManagementKey  string
	AdminKey       string
	DataKey        string
	DataKeyPath    string
	CollectorMode  string
	Queue          string
	PopSide        string
	BatchSize      int
	PollInterval   time.Duration
	QueryLimit     int
	PanelPath      string
	CORSOrigins    []string
	TLSSkipVerify  bool
}

type LoadOptions struct {
	CreateDefaultConfig bool
}

type fileConfig struct {
	HTTPAddr          string   `json:"httpAddr,omitempty"`
	DataDir           string   `json:"dataDir,omitempty"`
	DBPath            string   `json:"dbPath,omitempty"`
	CPAUpstreamURL    string   `json:"cpaUpstreamUrl,omitempty"`
	ManagementKey     string   `json:"managementKey,omitempty"`
	CPAManagementKey  string   `json:"cpaManagementKey,omitempty"`
	ManagementKeyFile string   `json:"managementKeyFile,omitempty"`
	AdminKey          string   `json:"adminKey,omitempty"`
	AdminPassword     string   `json:"adminPassword,omitempty"`
	AdminKeyFile      string   `json:"adminKeyFile,omitempty"`
	DataKey           string   `json:"dataKey,omitempty"`
	DataKeyFile       string   `json:"dataKeyFile,omitempty"`
	DataKeyPath       string   `json:"dataKeyPath,omitempty"`
	CollectorMode     string   `json:"collectorMode,omitempty"`
	Queue             string   `json:"queue,omitempty"`
	PopSide           string   `json:"popSide,omitempty"`
	BatchSize         int      `json:"batchSize,omitempty"`
	PollIntervalMS    int      `json:"pollIntervalMs,omitempty"`
	QueryLimit        int      `json:"queryLimit,omitempty"`
	PanelPath         string   `json:"panelPath,omitempty"`
	CORSOrigins       []string `json:"corsOrigins,omitempty"`
	TLSSkipVerify     bool     `json:"tlsSkipVerify,omitempty"`
}

func Load() (Config, error) {
	return LoadWithOptions(LoadOptions{CreateDefaultConfig: true})
}

func LoadWithoutCreatingDefault() (Config, error) {
	return LoadWithOptions(LoadOptions{})
}

func LoadWithOptions(options LoadOptions) (Config, error) {
	cfgFile, cfgDir, err := loadFileConfig(options)
	if err != nil {
		return Config{}, err
	}

	dataDirFallback := "/data"
	if cfgFile.DataDir != "" {
		dataDirFallback = resolveConfigPath(cfgFile.DataDir, cfgDir)
	} else if cfgDir != "" {
		dataDirFallback = resolveConfigPath("./data", cfgDir)
	}
	dataDir := env("USAGE_DATA_DIR", dataDirFallback)

	dbPathFallback := filepath.Join(dataDir, "usage.sqlite")
	if !hasEnv("USAGE_DATA_DIR") && cfgFile.DBPath != "" {
		dbPathFallback = resolveConfigPath(cfgFile.DBPath, cfgDir)
	}

	managementKeyFile := defaultSecretFile
	if cfgFile.ManagementKeyFile != "" {
		managementKeyFile = resolveConfigPath(cfgFile.ManagementKeyFile, cfgDir)
	}

	adminKeyFile := defaultAdminSecretFile
	if cfgFile.AdminKeyFile != "" {
		adminKeyFile = resolveConfigPath(cfgFile.AdminKeyFile, cfgDir)
	}

	dataKeyFile := defaultDataKeySecretFile
	if cfgFile.DataKeyFile != "" {
		dataKeyFile = resolveConfigPath(cfgFile.DataKeyFile, cfgDir)
	}
	dataKeyPath := resolveConfigPath(cfgFile.DataKeyPath, cfgDir)
	if dataKeyPath == "" {
		dataKeyPath = filepath.Join(dataDir, "data.key")
	}

	return Config{
		HTTPAddr:       env("HTTP_ADDR", stringFallback(cfgFile.HTTPAddr, "0.0.0.0:8318")),
		DataDir:        dataDir,
		DBPath:         env("USAGE_DB_PATH", dbPathFallback),
		CPAUpstreamURL: env("CPA_UPSTREAM_URL", cfgFile.CPAUpstreamURL),
		ManagementKey: readSecretWithConfigValue(
			"CPA_MANAGEMENT_KEY",
			"CPA_MANAGEMENT_KEY_FILE",
			managementKeyFile,
			firstNonEmpty(cfgFile.ManagementKey, cfgFile.CPAManagementKey),
		),
		AdminKey: readSecretWithConfigValue(
			"CPA_MANAGER_ADMIN_KEY",
			"CPA_MANAGER_ADMIN_KEY_FILE",
			adminKeyFile,
			firstNonEmpty(cfgFile.AdminKey, cfgFile.AdminPassword),
		),
		DataKey: readSecretWithConfigValue(
			"CPA_MANAGER_DATA_KEY",
			"CPA_MANAGER_DATA_KEY_FILE",
			dataKeyFile,
			cfgFile.DataKey,
		),
		DataKeyPath:    env("CPA_MANAGER_DATA_KEY_PATH", dataKeyPath),
		CollectorMode:  normalizeCollectorMode(env("USAGE_COLLECTOR_MODE", stringFallback(cfgFile.CollectorMode, "auto"))),
		Queue:          env("USAGE_RESP_QUEUE", stringFallback(cfgFile.Queue, "usage")),
		PopSide:        env("USAGE_RESP_POP_SIDE", stringFallback(cfgFile.PopSide, "right")),
		BatchSize:      envInt("USAGE_BATCH_SIZE", intFallback(cfgFile.BatchSize, 100)),
		PollInterval:   time.Duration(envInt("USAGE_POLL_INTERVAL_MS", intFallback(cfgFile.PollIntervalMS, 500))) * time.Millisecond,
		QueryLimit:     envInt("USAGE_QUERY_LIMIT", intFallback(cfgFile.QueryLimit, 50000)),
		PanelPath:      env("PANEL_PATH", resolveConfigPath(cfgFile.PanelPath, cfgDir)),
		CORSOrigins:    splitCSV(env("USAGE_CORS_ORIGINS", strings.Join(sliceFallback(cfgFile.CORSOrigins, []string{"*"}), ","))),
		TLSSkipVerify:  envBool("USAGE_RESP_TLS_SKIP_VERIFY", cfgFile.TLSSkipVerify),
	}, nil
}

func loadFileConfig(options LoadOptions) (fileConfig, string, error) {
	if configPath := strings.TrimSpace(os.Getenv(configEnvKey)); configPath != "" {
		if !options.CreateDefaultConfig {
			cfg, cfgDir, ok, err := readFileConfig(configPath)
			if err != nil || ok {
				return cfg, cfgDir, err
			}
			return fileConfig{}, filepath.Dir(configPath), nil
		}
		return readOrCreateFileConfig(configPath)
	}

	configPath, err := executableConfigPath()
	if err != nil {
		return fileConfig{}, "", err
	}
	cfg, cfgDir, ok, err := readFileConfig(configPath)
	if err != nil || ok {
		return cfg, cfgDir, err
	}
	if hasEnv("USAGE_DATA_DIR") || hasEnv("USAGE_DB_PATH") {
		return fileConfig{}, "", nil
	}
	if !options.CreateDefaultConfig {
		return fileConfig{}, filepath.Dir(configPath), nil
	}
	return createDefaultFileConfig(configPath)
}

func readOrCreateFileConfig(configPath string) (fileConfig, string, error) {
	cfg, cfgDir, ok, err := readFileConfig(configPath)
	if err != nil || ok {
		return cfg, cfgDir, err
	}
	return createDefaultFileConfig(configPath)
}

func readFileConfig(configPath string) (fileConfig, string, bool, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return fileConfig{}, filepath.Dir(configPath), false, nil
		}
		return fileConfig{}, filepath.Dir(configPath), false, fmt.Errorf("read config %s: %w", configPath, err)
	}
	var cfg fileConfig
	data = []byte(strings.TrimPrefix(string(data), "\ufeff"))
	if isYAMLConfig(configPath) {
		parsed, err := parseYAMLFileConfig(string(data))
		if err != nil {
			return fileConfig{}, filepath.Dir(configPath), false, fmt.Errorf("parse config %s: %w", configPath, err)
		}
		return parsed, filepath.Dir(configPath), true, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fileConfig{}, filepath.Dir(configPath), false, fmt.Errorf("parse config %s: %w", configPath, err)
	}
	return cfg, filepath.Dir(configPath), true, nil
}

func createDefaultFileConfig(configPath string) (fileConfig, string, error) {
	cfg := fileConfig{
		HTTPAddr:         "0.0.0.0:8318",
		DataDir:          "./data",
		DBPath:           "./data/usage.sqlite",
		CPAUpstreamURL:   "http://your-cpa-host:8317",
		CPAManagementKey: "admin",
		AdminPassword:    "admin",
		CollectorMode:    "auto",
		Queue:            "usage",
		PopSide:          "right",
		BatchSize:        100,
		PollIntervalMS:   500,
		QueryLimit:       50000,
		DataKeyPath:      "./data/data.key",
		CORSOrigins:      []string{"*"},
	}
	var data []byte
	var err error
	if isYAMLConfig(configPath) {
		data = []byte(defaultYAMLConfig())
	} else {
		data, err = json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			return fileConfig{}, "", err
		}
		data = append(data, '\n')
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return fileConfig{}, "", fmt.Errorf("create config directory %s: %w", filepath.Dir(configPath), err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fileConfig{}, "", fmt.Errorf("create default config %s: %w", configPath, err)
	}
	return cfg, filepath.Dir(configPath), nil
}

func executableConfigPath() (string, error) {
	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable path: %w", err)
	}
	dir := filepath.Dir(executable)
	legacyPath := filepath.Join(dir, legacyDefaultConfigName)
	if _, err := os.Stat(legacyPath); err == nil {
		return legacyPath, nil
	}
	return filepath.Join(dir, defaultConfigName), nil
}

func isYAMLConfig(configPath string) bool {
	ext := strings.ToLower(filepath.Ext(configPath))
	return ext == ".yaml" || ext == ".yml"
}

func defaultYAMLConfig() string {
	return `server:
  listen: "0.0.0.0:8318"
  dataDir: "./data"
  dbPath: "./data/usage.sqlite"
admin:
  password: "admin"
cpa:
  url: "http://your-cpa-host:8317"
  password: "admin"
collector:
  mode: "auto"
  queue: "usage"
  popSide: "right"
  batchSize: 100
  pollIntervalMs: 500
  queryLimit: 50000
security:
  dataKeyPath: "./data/data.key"
  tlsSkipVerify: false
cors:
  origins: ["*"]
`
}

func parseYAMLFileConfig(raw string) (fileConfig, error) {
	values := map[string]string{}
	section := ""
	for lineNumber, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, " \t\r")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent%2 != 0 {
			return fileConfig{}, fmt.Errorf("line %d: indentation must use multiples of two spaces", lineNumber+1)
		}
		trimmed := strings.TrimSpace(stripYAMLComment(line))
		if trimmed == "" {
			continue
		}
		key, value, ok := strings.Cut(trimmed, ":")
		if !ok {
			return fileConfig{}, fmt.Errorf("line %d: expected key: value", lineNumber+1)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return fileConfig{}, fmt.Errorf("line %d: empty key", lineNumber+1)
		}
		if value == "" {
			if indent == 0 {
				section = key
				continue
			}
			return fileConfig{}, fmt.Errorf("line %d: nested sections deeper than one level are not supported", lineNumber+1)
		}
		path := key
		if indent > 0 && section != "" {
			path = section + "." + key
		}
		values[path] = parseYAMLScalar(value)
	}

	return yamlValuesToFileConfig(values), nil
}

func stripYAMLComment(line string) string {
	inSingle := false
	inDouble := false
	escaped := false
	for i, r := range line {
		switch {
		case escaped:
			escaped = false
		case r == '\\' && inDouble:
			escaped = true
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case r == '#' && !inSingle && !inDouble:
			return line[:i]
		}
	}
	return line
}

func parseYAMLScalar(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			if unquoted, err := strconv.Unquote(value); err == nil {
				return unquoted
			}
			return strings.Trim(value[1:len(value)-1], " ")
		}
	}
	return value
}

func yamlValuesToFileConfig(values map[string]string) fileConfig {
	var cfg fileConfig
	read := func(keys ...string) string {
		for _, key := range keys {
			if value := strings.TrimSpace(values[key]); value != "" {
				return value
			}
		}
		return ""
	}
	cfg.HTTPAddr = read("server.listen", "httpAddr", "listen")
	cfg.DataDir = read("server.dataDir", "dataDir")
	cfg.DBPath = read("server.dbPath", "dbPath")
	cfg.CPAUpstreamURL = read("cpa.url", "cpaUpstreamUrl", "cpaUpstreamURL")
	cfg.ManagementKey = read("managementKey")
	cfg.CPAManagementKey = read("cpa.password", "cpa.managementKey", "cpaManagementKey")
	cfg.ManagementKeyFile = read("cpa.passwordFile", "cpa.managementKeyFile", "managementKeyFile")
	cfg.AdminKey = read("admin.key", "adminKey")
	cfg.AdminPassword = read("admin.password", "adminPassword")
	cfg.AdminKeyFile = read("admin.keyFile", "adminKeyFile")
	cfg.DataKey = read("security.dataKey", "dataKey")
	cfg.DataKeyFile = read("security.dataKeyFile", "dataKeyFile")
	cfg.DataKeyPath = read("security.dataKeyPath", "dataKeyPath")
	cfg.CollectorMode = read("collector.mode", "collectorMode")
	cfg.Queue = read("collector.queue", "queue")
	cfg.PopSide = read("collector.popSide", "popSide")
	cfg.BatchSize = parsePositiveInt(read("collector.batchSize", "batchSize"))
	cfg.PollIntervalMS = parsePositiveInt(read("collector.pollIntervalMs", "pollIntervalMs"))
	cfg.QueryLimit = parsePositiveInt(read("collector.queryLimit", "queryLimit"))
	cfg.PanelPath = read("panel.path", "panelPath")
	cfg.CORSOrigins = parseYAMLStringList(read("cors.origins", "corsOrigins"))
	cfg.TLSSkipVerify = parseBool(read("security.tlsSkipVerify", "tlsSkipVerify"))
	return cfg
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseYAMLStringList(value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
		value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
		if value == "" {
			return nil
		}
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, part := range parts {
			if item := parseYAMLScalar(strings.TrimSpace(part)); item != "" {
				result = append(result, item)
			}
		}
		return result
	}
	return splitCSV(value)
}

func normalizeCollectorMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "http", "resp", "subscribe":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return "auto"
	}
}

func hasEnv(key string) bool {
	return strings.TrimSpace(os.Getenv(key)) != ""
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		return fallback
	}
	return parsed
}

func envBool(key string, fallback bool) bool {
	value := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if value == "" {
		return fallback
	}
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

func stringFallback(value string, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func intFallback(value int, fallback int) int {
	if value <= 0 {
		return fallback
	}
	return value
}

func sliceFallback(value []string, fallback []string) []string {
	if len(value) == 0 {
		return fallback
	}
	return value
}

func resolveConfigPath(path string, baseDir string) string {
	path = strings.TrimSpace(path)
	if path == "" || filepath.IsAbs(path) || baseDir == "" {
		return path
	}
	return filepath.Join(baseDir, path)
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func readSecret(envKey string, fileEnvKey string, defaultFile string) string {
	return readSecretWithConfigValue(envKey, fileEnvKey, defaultFile, "")
}

func readSecretWithConfigValue(envKey string, fileEnvKey string, defaultFile string, configValue string) string {
	if value := strings.TrimSpace(os.Getenv(envKey)); value != "" {
		return value
	}
	if value := strings.TrimSpace(configValue); value != "" {
		return value
	}

	path := strings.TrimSpace(os.Getenv(fileEnvKey))
	if path == "" {
		path = defaultFile
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
