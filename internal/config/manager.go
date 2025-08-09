package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/BurntSushi/toml"
	"github.com/fsnotify/fsnotify"
	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
	"github.com/spf13/viper"
)

// Manager implements the ConfigManager interface
type Manager struct {
	mu         sync.RWMutex
	viper      *viper.Viper
	config     *internal.Config
	watcher    *fsnotify.Watcher
	configPath string
	logger     *logging.Logger
}

// NewManager creates a new configuration manager
func NewManager() *Manager {
	return &Manager{
		viper:  viper.New(),
		logger: logging.NewLogger("info"),
	}
}

// NewManagerWithLevel creates a new configuration manager with a configurable log level.
// Pass level "none" in tests to silence logs.
func NewManagerWithLevel(level string) *Manager {
	return &Manager{
		viper:  viper.New(),
		logger: logging.NewLogger(level),
	}
}

// Load loads configuration from file or returns default configuration
func (m *Manager) Load() (*internal.Config, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.logger.Debug("Loading configuration", map[string]interface{}{
		"config_path": m.configPath,
	})

	// If a specific config path is set, try to load it directly
	if m.configPath != "" {
		if _, err := os.Stat(m.configPath); err == nil {
			m.logger.Debug("Loading configuration from specified path", map[string]interface{}{
				"path": m.configPath,
			})

			data, err := os.ReadFile(m.configPath)
			if err != nil {
				m.logger.Error("Failed to read config file", err, map[string]interface{}{
					"path": m.configPath,
				})
				return nil, fmt.Errorf("failed to read config file: %w", err)
			}

			var config internal.Config
			if err := toml.Unmarshal(data, &config); err != nil {
				m.logger.Error("Failed to unmarshal config", err, map[string]interface{}{
					"path": m.configPath,
				})
				return nil, fmt.Errorf("failed to unmarshal config: %w", err)
			}

			if err := m.validateConfig(&config); err != nil {
				m.logger.Error("Config validation failed", err, map[string]interface{}{
					"path": m.configPath,
				})
				return nil, fmt.Errorf("config validation failed: %w", err)
			}

			m.config = &config
			m.logger.Info("Configuration loaded successfully", map[string]interface{}{
				"path":        m.configPath,
				"device_name": config.Device.Name,
				"device_role": config.Device.Role,
			})
			return &config, nil
		} else {
			m.logger.Warn("Specified config file does not exist", map[string]interface{}{
				"path": m.configPath,
			})
		}
	}

	m.setupViper()

	if err := m.viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			m.logger.Info("No config file found, using default configuration", nil)
			config := internal.DefaultConfig()
			m.config = config
			return config, nil
		}
		m.logger.Error("Failed to read config file", err, nil)
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	configFile := m.viper.ConfigFileUsed()
	m.logger.Debug("Found config file", map[string]interface{}{
		"path": configFile,
	})

	data, err := os.ReadFile(configFile)
	if err != nil {
		m.logger.Error("Failed to read config file", err, map[string]interface{}{
			"path": configFile,
		})
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config internal.Config
	if err := toml.Unmarshal(data, &config); err != nil {
		m.logger.Error("Failed to unmarshal config", err, map[string]interface{}{
			"path": configFile,
		})
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if err := m.validateConfig(&config); err != nil {
		m.logger.Error("Config validation failed", err, map[string]interface{}{
			"path": configFile,
		})
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	m.config = &config
	m.logger.Info("Configuration loaded successfully", map[string]interface{}{
		"path":        configFile,
		"device_name": config.Device.Name,
		"device_role": config.Device.Role,
	})
	return &config, nil
}

// Save saves configuration to file
func (m *Manager) Save(config *internal.Config) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	configPath := m.getConfigPath()
	m.logger.Debug("Saving configuration", map[string]interface{}{
		"path":        configPath,
		"device_name": config.Device.Name,
	})

	configDir := filepath.Dir(configPath)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		m.logger.Error("Failed to create config directory", err, map[string]interface{}{
			"directory": configDir,
		})
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := toml.Marshal(config)
	if err != nil {
		m.logger.Error("Failed to marshal config", err, nil)
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		m.logger.Error("Failed to write config file", err, map[string]interface{}{
			"path": configPath,
		})
		return fmt.Errorf("failed to write config file: %w", err)
	}

	m.config = config
	m.logger.Info("Configuration saved successfully", map[string]interface{}{
		"path":        configPath,
		"device_name": config.Device.Name,
	})
	return nil
}

// Watch watches for configuration file changes
func (m *Manager) Watch(ctx context.Context) (<-chan *internal.Config, error) {
	configChan := make(chan *internal.Config, 1)

	m.logger.Debug("Starting configuration file watcher", nil)

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		m.logger.Error("Failed to create file watcher", err, nil)
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	m.mu.Lock()
	m.watcher = watcher
	m.mu.Unlock()

	configDir := filepath.Dir(m.getConfigPath())
	if err := watcher.Add(configDir); err != nil {
		_ = watcher.Close()
		m.logger.Error("Failed to watch config directory", err, map[string]interface{}{
			"directory": configDir,
		})
		return nil, fmt.Errorf("failed to watch config directory: %w", err)
	}

	m.logger.Info("Configuration file watcher started", map[string]interface{}{
		"directory": configDir,
	})

	go func() {
		defer func() { _ = watcher.Close() }()
		defer close(configChan)
		defer m.logger.Debug("Configuration file watcher stopped", nil)

		for {
			select {
			case <-ctx.Done():
				m.logger.Debug("Configuration file watcher context cancelled", nil)
				return
			case event, ok := <-watcher.Events:
				if !ok {
					m.logger.Debug("Configuration file watcher events channel closed", nil)
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					if strings.HasSuffix(event.Name, ".toml") || strings.HasSuffix(event.Name, ".ini") {
						m.logger.Debug("Configuration file changed, reloading", map[string]interface{}{
							"file":      event.Name,
							"operation": event.Op.String(),
						})
						if config, err := m.Load(); err == nil {
							select {
							case configChan <- config:
								m.logger.Info("Configuration reloaded and sent to channel", map[string]interface{}{
									"file":        event.Name,
									"device_name": config.Device.Name,
								})
							case <-ctx.Done():
								return
							}
						} else {
							m.logger.Error("Failed to reload configuration", err, map[string]interface{}{
								"file": event.Name,
							})
						}
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					m.logger.Debug("Configuration file watcher errors channel closed", nil)
					return
				}
				m.logger.Error("Config watcher error", err, nil)
			}
		}
	}()

	return configChan, nil
}

func (m *Manager) setupViper() {
	m.viper.SetConfigName("config")
	m.viper.SetConfigType("toml")

	for _, path := range m.getConfigPaths() {
		m.viper.AddConfigPath(path)
	}

	m.viper.SetEnvPrefix("MeshExec")
	m.viper.AutomaticEnv()
	m.setDefaults()
}

func (m *Manager) getConfigPaths() []string {
	var paths []string
	paths = append(paths, ".")
	if userConfigDir, err := m.getUserConfigDir(); err == nil {
		paths = append(paths, userConfigDir)
	}
	if systemConfigDir := m.getSystemConfigDir(); systemConfigDir != "" {
		paths = append(paths, systemConfigDir)
	}
	return paths
}

func (m *Manager) getConfigPath() string {
	if m.configPath != "" {
		return m.configPath
	}
	if userConfigDir, err := m.getUserConfigDir(); err == nil {
		return filepath.Join(userConfigDir, "config.toml")
	}
	return "config.toml"
}

func (m *Manager) getUserConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		return filepath.Join(appData, "meshexec"), nil
	default:
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(homeDir, ".meshexec"), nil
	}
}

func (m *Manager) getSystemConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		programData := os.Getenv("PROGRAMDATA")
		if programData == "" {
			return ""
		}
		return filepath.Join(programData, "meshexec")
	default:
		return "/etc/meshexec"
	}
}

func (m *Manager) setDefaults() {
	m.viper.SetDefault("device.name", "meshexec-device")
	m.viper.SetDefault("device.role", "worker")
	m.viper.SetDefault("device.os", runtime.GOOS)
	m.viper.SetDefault("device.arch", runtime.GOARCH)
	m.viper.SetDefault("device.location", "unknown")
	m.viper.SetDefault("device.zone", "default")

	m.viper.SetDefault("security.require_auth", true)
	m.viper.SetDefault("security.private_key_path", "~/.meshexec/private.key")
	m.viper.SetDefault("security.public_key_path", "~/.meshexec/public.key")

	m.viper.SetDefault("network.service_uuid", "12345678-1234-1234-1234-123456789abc")
	m.viper.SetDefault("network.characteristic_uuid", "87654321-4321-4321-4321-cba987654321")
	m.viper.SetDefault("network.advertise_interval", 1000)
	m.viper.SetDefault("network.scan_interval", 1000)
	m.viper.SetDefault("network.connection_timeout", 5000)
	m.viper.SetDefault("network.max_peers", 10)
	m.viper.SetDefault("network.ttl", 5)

	m.viper.SetDefault("safety.safe_mode", true)
	m.viper.SetDefault("safety.dangerous_commands", []string{"rm -rf", "del /s", "format", "dd if="})
	m.viper.SetDefault("safety.max_command_length", 1024)
	m.viper.SetDefault("safety.execution_timeout", 30000)

	m.viper.SetDefault("logging.level", "info")
	m.viper.SetDefault("logging.format", "json")
	m.viper.SetDefault("logging.output", "stdout")
	m.viper.SetDefault("logging.max_size", 100)
	m.viper.SetDefault("logging.max_backups", 3)
	m.viper.SetDefault("logging.max_age", 28)
	m.viper.SetDefault("logging.compress", true)
}

func (m *Manager) validateConfig(config *internal.Config) error {
	if config.Device.Name == "" {
		return fmt.Errorf("device name cannot be empty")
	}
	if config.Network.ServiceUUID == "" {
		return fmt.Errorf("service UUID cannot be empty")
	}
	if config.Network.TTL <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}
	if config.Network.MaxPeers <= 0 {
		return fmt.Errorf("max peers must be greater than 0")
	}
	if config.Safety.MaxCommandLength <= 0 {
		return fmt.Errorf("max command length must be greater than 0")
	}
	if config.Safety.ExecutionTimeout <= 0 {
		return fmt.Errorf("execution timeout must be greater than 0")
	}
	return nil
}

func (m *Manager) SetConfigPath(path string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.configPath = path
}

func (m *Manager) GetConfigPath() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.getConfigPath()
}

func (m *Manager) CreateDefaultConfig() error {
	m.logger.Info("Creating default configuration file", nil)
	config := internal.DefaultConfig()
	return m.Save(config)
}