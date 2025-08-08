package internal

// Config represents the main configuration structure
type Config struct {
	Device   DeviceConfig   `toml:"device" ini:"device"`
	Security SecurityConfig `toml:"security" ini:"security"`
	Network  NetworkConfig  `toml:"network" ini:"network"`
	Safety   SafetyConfig   `toml:"safety" ini:"safety"`
	Logging  LoggingConfig  `toml:"logging" ini:"logging"`
}

// DeviceConfig represents device-specific configuration
type DeviceConfig struct {
	Name     string            `toml:"name" ini:"name"`
	Role     string            `toml:"role" ini:"role"`
	Tags     map[string]string `toml:"tags" ini:"tags"`
	OS       string            `toml:"os" ini:"os"`
	Arch     string            `toml:"arch" ini:"arch"`
	Location string            `toml:"location" ini:"location"`
	Zone     string            `toml:"zone" ini:"zone"`
}

// SecurityConfig represents security-related configuration
type SecurityConfig struct {
	PrivateKeyPath string   `toml:"private_key_path" ini:"private_key_path"`
	PublicKeyPath  string   `toml:"public_key_path" ini:"public_key_path"`
	EncryptionKey  string   `toml:"encryption_key" ini:"encryption_key"`
	AllowedSenders []string `toml:"allowed_senders" ini:"allowed_senders"`
	DeniedSenders  []string `toml:"denied_senders" ini:"denied_senders"`
	RequireAuth    bool     `toml:"require_auth" ini:"require_auth"`
}

// NetworkConfig represents network-related configuration
type NetworkConfig struct {
	ServiceUUID        string `toml:"service_uuid" ini:"service_uuid"`
	CharacteristicUUID string `toml:"characteristic_uuid" ini:"characteristic_uuid"`
	AdvertiseInterval  int    `toml:"advertise_interval" ini:"advertise_interval"`
	ScanInterval       int    `toml:"scan_interval" ini:"scan_interval"`
	ConnectionTimeout  int    `toml:"connection_timeout" ini:"connection_timeout"`
	MaxPeers           int    `toml:"max_peers" ini:"max_peers"`
	TTL                int    `toml:"ttl" ini:"ttl"`
}

// SafetyConfig represents safety-related configuration
type SafetyConfig struct {
	SafeMode          bool     `toml:"safe_mode" ini:"safe_mode"`
	DangerousCommands []string `toml:"dangerous_commands" ini:"dangerous_commands"`
	AllowedCommands   []string `toml:"allowed_commands" ini:"allowed_commands"`
	DeniedCommands    []string `toml:"denied_commands" ini:"denied_commands"`
	MaxCommandLength  int      `toml:"max_command_length" ini:"max_command_length"`
	ExecutionTimeout  int      `toml:"execution_timeout" ini:"execution_timeout"`
}

// LoggingConfig represents logging-related configuration
type LoggingConfig struct {
	Level      string `toml:"level" ini:"level"`
	Format     string `toml:"format" ini:"format"`
	Output     string `toml:"output" ini:"output"`
	MaxSize    int    `toml:"max_size" ini:"max_size"`
	MaxBackups int    `toml:"max_backups" ini:"max_backups"`
	MaxAge     int    `toml:"max_age" ini:"max_age"`
	Compress   bool   `toml:"compress" ini:"compress"`
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		Device: DeviceConfig{
			Name:     "meshexec-device",
			Role:     "worker",
			Tags:     make(map[string]string),
			OS:       "unknown",
			Arch:     "unknown",
			Location: "unknown",
			Zone:     "default",
		},
		Security: SecurityConfig{
			PrivateKeyPath: "~/.meshexec/private.key",
			PublicKeyPath:  "~/.meshexec/public.key",
			EncryptionKey:  "",
			AllowedSenders: []string{},
			DeniedSenders:  []string{},
			RequireAuth:    true,
		},
		Network: NetworkConfig{
			ServiceUUID:        "12345678-1234-1234-1234-123456789abc",
			CharacteristicUUID: "87654321-4321-4321-4321-cba987654321",
			AdvertiseInterval:  1000,
			ScanInterval:       1000,
			ConnectionTimeout:  5000,
			MaxPeers:           10,
			TTL:                5,
		},
		Safety: SafetyConfig{
			SafeMode:          true,
			DangerousCommands: []string{"rm -rf", "del /s", "format", "dd if="},
			AllowedCommands:   []string{},
			DeniedCommands:    []string{},
			MaxCommandLength:  1024,
			ExecutionTimeout:  30000,
		},
		Logging: LoggingConfig{
			Level:      "info",
			Format:     "json",
			Output:     "stdout",
			MaxSize:    100,
			MaxBackups: 3,
			MaxAge:     28,
			Compress:   true,
		},
	}
}
