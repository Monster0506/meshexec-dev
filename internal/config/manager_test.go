package config

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManager(t *testing.T) {
	manager := NewManagerWithLevel("none")
	assert.NotNil(t, manager)
	assert.NotNil(t, manager.viper)
}

func TestManager_Load_DefaultConfig(t *testing.T) {
	manager := NewManagerWithLevel("none")

	// Test loading when no config file exists (should return default)
	config, err := manager.Load()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Verify default values
	assert.Equal(t, "meshexec-device", config.Device.Name)
	assert.Equal(t, "worker", config.Device.Role)
	assert.Equal(t, "unknown", config.Device.OS)
	assert.Equal(t, "unknown", config.Device.Arch)
	assert.True(t, config.Security.RequireAuth)
	assert.Equal(t, 5, config.Network.TTL)
	assert.Equal(t, 10, config.Network.MaxPeers)
	assert.True(t, config.Safety.SafeMode)
}

func TestManager_Load_FromFile(t *testing.T) {
	// Create temporary config file
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	configData := `
[device]
name = "test-device"
role = "test-role"
os = "linux"
arch = "amd64"

[security]
require_auth = false

[network]
service_uuid = "test-uuid"
ttl = 10
max_peers = 20

[safety]
safe_mode = false
max_command_length = 1024
execution_timeout = 30000
`

	err := os.WriteFile(configPath, []byte(configData), 0644)
	require.NoError(t, err)

	manager := NewManagerWithLevel("none")
	manager.SetConfigPath(configPath)

	config, err := manager.Load()
	require.NoError(t, err)
	assert.NotNil(t, config)

	// Verify loaded values
	assert.Equal(t, "test-device", config.Device.Name)
	assert.Equal(t, "test-role", config.Device.Role)
	assert.Equal(t, "linux", config.Device.OS)
	assert.Equal(t, "amd64", config.Device.Arch)
	assert.False(t, config.Security.RequireAuth)
	assert.Equal(t, 10, config.Network.TTL)
	assert.Equal(t, 20, config.Network.MaxPeers)
	assert.False(t, config.Safety.SafeMode)
}

func TestManager_Save(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	manager := NewManagerWithLevel("none")
	manager.SetConfigPath(configPath)

	// Create custom config
	config := &internal.Config{
		Device: internal.DeviceConfig{
			Name: "custom-device",
			Role: "custom-role",
			OS:   "windows",
			Arch: "arm64",
		},
		Security: internal.SecurityConfig{
			RequireAuth: false,
		},
		Network: internal.NetworkConfig{
			ServiceUUID: "custom-uuid",
			TTL:         15,
			MaxPeers:    25,
		},
		Safety: internal.SafetyConfig{
			SafeMode:         false,
			MaxCommandLength: 1024,
			ExecutionTimeout: 30000,
		},
	}

	// Save config
	err := manager.Save(config)
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Load config back and verify
	loadedConfig, err := manager.Load()
	require.NoError(t, err)
	assert.Equal(t, "custom-device", loadedConfig.Device.Name)
	assert.Equal(t, "custom-role", loadedConfig.Device.Role)
	assert.Equal(t, "windows", loadedConfig.Device.OS)
	assert.Equal(t, "arm64", loadedConfig.Device.Arch)
	assert.False(t, loadedConfig.Security.RequireAuth)
	assert.Equal(t, 15, loadedConfig.Network.TTL)
	assert.Equal(t, 25, loadedConfig.Network.MaxPeers)
	assert.False(t, loadedConfig.Safety.SafeMode)
}

func TestManager_ValidateConfig(t *testing.T) {
	manager := NewManagerWithLevel("none")

	tests := []struct {
		name    string
		config  *internal.Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "test-device",
				},
				Network: internal.NetworkConfig{
					ServiceUUID: "test-uuid",
					TTL:         5,
					MaxPeers:    10,
				},
				Safety: internal.SafetyConfig{
					MaxCommandLength: 1024,
					ExecutionTimeout: 30000,
				},
			},
			wantErr: false,
		},
		{
			name: "empty device name",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "",
				},
			},
			wantErr: true,
		},
		{
			name: "empty service UUID",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "test-device",
				},
				Network: internal.NetworkConfig{
					ServiceUUID: "",
				},
			},
			wantErr: true,
		},
		{
			name: "invalid TTL",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "test-device",
				},
				Network: internal.NetworkConfig{
					ServiceUUID: "test-uuid",
					TTL:         0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max peers",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "test-device",
				},
				Network: internal.NetworkConfig{
					ServiceUUID: "test-uuid",
					TTL:         5,
					MaxPeers:    0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid max command length",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "test-device",
				},
				Network: internal.NetworkConfig{
					ServiceUUID: "test-uuid",
					TTL:         5,
					MaxPeers:    10,
				},
				Safety: internal.SafetyConfig{
					MaxCommandLength: 0,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid execution timeout",
			config: &internal.Config{
				Device: internal.DeviceConfig{
					Name: "test-device",
				},
				Network: internal.NetworkConfig{
					ServiceUUID: "test-uuid",
					TTL:         5,
					MaxPeers:    10,
				},
				Safety: internal.SafetyConfig{
					MaxCommandLength: 1024,
					ExecutionTimeout: 0,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.validateConfig(tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_Watch(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	manager := NewManagerWithLevel("none")
	manager.SetConfigPath(configPath)

	// Create initial config
	initialConfig := &internal.Config{
		Device: internal.DeviceConfig{
			Name: "initial-device",
		},
		Network: internal.NetworkConfig{
			ServiceUUID: "test-uuid",
			TTL:         5,
			MaxPeers:    10,
		},
		Safety: internal.SafetyConfig{
			MaxCommandLength: 1024,
			ExecutionTimeout: 30000,
		},
	}

	err := manager.Save(initialConfig)
	require.NoError(t, err)

	// Start watching
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	configChan, err := manager.Watch(ctx)
	require.NoError(t, err)

	// Wait a bit for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	updatedConfig := &internal.Config{
		Device: internal.DeviceConfig{
			Name: "updated-device",
		},
		Network: internal.NetworkConfig{
			ServiceUUID: "test-uuid",
			TTL:         5,
			MaxPeers:    10,
		},
		Safety: internal.SafetyConfig{
			MaxCommandLength: 1024,
			ExecutionTimeout: 30000,
		},
	}

	err = manager.Save(updatedConfig)
	require.NoError(t, err)

	// Wait for config change
	select {
	case config := <-configChan:
		assert.Equal(t, "updated-device", config.Device.Name)
	case <-ctx.Done():
		t.Fatal("timeout waiting for config change")
	}
}

func TestManager_CreateDefaultConfig(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	manager := NewManagerWithLevel("none")
	manager.SetConfigPath(configPath)

	// Create default config
	err := manager.CreateDefaultConfig()
	require.NoError(t, err)

	// Verify file was created
	_, err = os.Stat(configPath)
	assert.NoError(t, err)

	// Load and verify default values
	config, err := manager.Load()
	require.NoError(t, err)
	assert.Equal(t, "meshexec-device", config.Device.Name)
	assert.Equal(t, "worker", config.Device.Role)
	assert.True(t, config.Security.RequireAuth)
	assert.Equal(t, 5, config.Network.TTL)
	assert.Equal(t, 10, config.Network.MaxPeers)
	assert.True(t, config.Safety.SafeMode)
}

func TestManager_GetConfigPaths(t *testing.T) {
	manager := NewManagerWithLevel("none")
	paths := manager.getConfigPaths()

	// Should include current directory
	assert.Contains(t, paths, ".")

	// Should include user config directory if available
	userConfigDir, err := manager.getUserConfigDir()
	if err == nil {
		assert.Contains(t, paths, userConfigDir)
	}

	// Should include system config directory if available
	systemConfigDir := manager.getSystemConfigDir()
	if systemConfigDir != "" {
		assert.Contains(t, paths, systemConfigDir)
	}
}

func TestManager_GetUserConfigDir(t *testing.T) {
	manager := NewManagerWithLevel("none")

	userConfigDir, err := manager.getUserConfigDir()
	if err != nil {
		// Skip test if we can't get user config dir
		t.Skip("Cannot get user config directory")
	}

	assert.NotEmpty(t, userConfigDir)
	assert.Contains(t, userConfigDir, "meshexec")
}

func TestManager_GetSystemConfigDir(t *testing.T) {
	manager := NewManagerWithLevel("none")

	systemConfigDir := manager.getSystemConfigDir()
	// System config dir might be empty on some systems, which is OK
	if systemConfigDir != "" {
		assert.Contains(t, systemConfigDir, "meshexec")
	}
}

func TestManager_CrossPlatformConfigLoading(t *testing.T) {
	manager := NewManagerWithLevel("none")

	// Test that config paths are correctly determined for different platforms
	paths := manager.getConfigPaths()
	assert.NotEmpty(t, paths)

	// Should always include current directory
	assert.Contains(t, paths, ".")

	// Should include user config directory
	userConfigDir, err := manager.getUserConfigDir()
	if err == nil {
		assert.Contains(t, paths, userConfigDir)
	}

	// Should include system config directory if available
	systemConfigDir := manager.getSystemConfigDir()
	if systemConfigDir != "" {
		assert.Contains(t, paths, systemConfigDir)
	}
}

func TestManager_CrossPlatformFileWatching(t *testing.T) {
	// Test file watching on different platforms
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	manager := NewManagerWithLevel("none")
	manager.SetConfigPath(configPath)

	// Create initial config
	initialConfig := &internal.Config{
		Device: internal.DeviceConfig{
			Name: "test-device",
		},
		Network: internal.NetworkConfig{
			ServiceUUID: "test-uuid",
			TTL:         5,
			MaxPeers:    10,
		},
		Safety: internal.SafetyConfig{
			MaxCommandLength: 1024,
			ExecutionTimeout: 30000,
		},
	}

	err := manager.Save(initialConfig)
	require.NoError(t, err)

	// Test that file watching works on this platform
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	configChan, err := manager.Watch(ctx)
	require.NoError(t, err)

	// Wait for watcher to start
	time.Sleep(100 * time.Millisecond)

	// Modify config file
	updatedConfig := &internal.Config{
		Device: internal.DeviceConfig{
			Name: "updated-device",
		},
		Network: internal.NetworkConfig{
			ServiceUUID: "updated-uuid",
			TTL:         10,
			MaxPeers:    20,
		},
		Safety: internal.SafetyConfig{
			MaxCommandLength: 2048,
			ExecutionTimeout: 60000,
		},
	}

	err = manager.Save(updatedConfig)
	require.NoError(t, err)

	// Wait for config change notification
	select {
	case receivedConfig := <-configChan:
		assert.Equal(t, "updated-device", receivedConfig.Device.Name)
		assert.Equal(t, "updated-uuid", receivedConfig.Network.ServiceUUID)
		assert.Equal(t, 10, receivedConfig.Network.TTL)
		assert.Equal(t, 20, receivedConfig.Network.MaxPeers)
	case <-ctx.Done():
		t.Fatal("Timeout waiting for config change notification")
	}
}

func TestManager_PlatformSpecificPathHandling(t *testing.T) {
	manager := NewManagerWithLevel("none")

	// Test Windows-specific path handling
	if runtime.GOOS == "windows" {
		// Test APPDATA environment variable handling
		appData := os.Getenv("APPDATA")
		if appData != "" {
			userConfigDir, err := manager.getUserConfigDir()
			require.NoError(t, err)
			assert.Contains(t, userConfigDir, appData)
			assert.Contains(t, userConfigDir, "meshexec")
		}

		// Test PROGRAMDATA environment variable handling
		programData := os.Getenv("PROGRAMDATA")
		if programData != "" {
			systemConfigDir := manager.getSystemConfigDir()
			assert.Contains(t, systemConfigDir, programData)
			assert.Contains(t, systemConfigDir, "meshexec")
		}
	} else {
		// Test Unix-specific path handling
		homeDir, err := os.UserHomeDir()
		require.NoError(t, err)

		userConfigDir, err := manager.getUserConfigDir()
		require.NoError(t, err)
		assert.Contains(t, userConfigDir, homeDir)
		assert.Contains(t, userConfigDir, ".meshexec")

		systemConfigDir := manager.getSystemConfigDir()
		assert.Equal(t, "/etc/meshexec", systemConfigDir)
	}
}

func TestManager_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.toml")

	manager := NewManagerWithLevel("none")
	manager.SetConfigPath(configPath)

	// Test that config file is created with appropriate permissions
	config := &internal.Config{
		Device: internal.DeviceConfig{
			Name: "test-device",
		},
		Network: internal.NetworkConfig{
			ServiceUUID: "test-uuid",
			TTL:         5,
			MaxPeers:    10,
		},
		Safety: internal.SafetyConfig{
			MaxCommandLength: 1024,
			ExecutionTimeout: 30000,
		},
	}

	err := manager.Save(config)
	require.NoError(t, err)

	// Check file permissions
	info, err := os.Stat(configPath)
	require.NoError(t, err)

	// Should be readable by owner
	assert.True(t, info.Mode().IsRegular())

	// On Unix systems, check specific permissions
	if runtime.GOOS != "windows" {
		// Should be readable and writable by owner (0644)
		expectedMode := os.FileMode(0644)
		assert.Equal(t, expectedMode, info.Mode().Perm())
	}
}

// Consolidated from manager_extra_test.go
func TestManager_GetConfigPath_Defaults(t *testing.T) {
	m := NewManagerWithLevel("none")
	if p := m.GetConfigPath(); p == "" {
		t.Fatalf("expected non-empty default config path")
	}
}
