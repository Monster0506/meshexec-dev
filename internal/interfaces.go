package internal

import (
	"context"
)

// MeshNode interface handles Bluetooth LE communication, message routing, and network topology management
type MeshNode interface {
	Start(ctx context.Context) error
	Stop() error
	SendMessage(msg *MeshMessage) error
	Subscribe(msgType MessageType) <-chan *MeshMessage
	GetPeers() []PeerInfo
}

// BLETransport interface handles low-level Bluetooth LE operations
type BLETransport interface {
	Advertise(ctx context.Context, serviceData []byte) error
	Scan(ctx context.Context) (<-chan *Advertisement, error)
	Connect(ctx context.Context, addr string) (*Connection, error)
	CreateGATTService() (*GATTService, error)
}

// Agent interface is the core service that processes commands, manages execution, and handles security
type Agent interface {
	Start(ctx context.Context) error
	Stop() error
	ProcessCommand(msg *MeshMessage) error
	ExecuteCommand(cmd string) (*ExecutionResult, error)
	ValidateCommand(msg *MeshMessage) error
}

// CommandExecutor interface handles command execution with cross-platform shell integration
type CommandExecutor interface {
	Execute(ctx context.Context, cmd string) (*ExecutionResult, error)
	ValidateCommand(cmd string) error
}

// SecurityManager interface handles cryptographic operations
type SecurityManager interface {
	SignMessage(msg *MeshMessage) error
	VerifySignature(msg *MeshMessage) error
	EncryptPayload(payload []byte) ([]byte, error)
	DecryptPayload(encrypted []byte) ([]byte, error)
}

// ConfigManager interface handles configuration file parsing and device settings management
type ConfigManager interface {
	Load() (*Config, error)
	Save(config *Config) error
	Watch(ctx context.Context) (<-chan *Config, error)
}

// TargetEvaluator interface evaluates device targeting expressions for command routing
type TargetEvaluator interface {
	Evaluate(expression string, device *DeviceInfo) (bool, error)
	Parse(expression string) (*TargetAST, error)
}

// CommandRunner interface provides user interface for command execution and network management
type CommandRunner interface {
	RunCommand(ctx context.Context, cmd string, target string, options RunOptions) (*ExecutionResults, error)
	RunDryRun(cmd string, target string) (*DryRunResults, error)
}

// NetworkManager interface handles network operations
type NetworkManager interface {
	JoinMesh(ctx context.Context) error
	LeaveMesh() error
	ListPeers() ([]PeerInfo, error)
	GetStatus() (*NetworkStatus, error)
}

// TUIManager interface handles terminal UI operations
type TUIManager interface {
	StartTUI(ctx context.Context) error
	UpdateResults(results *ExecutionResults)
	UpdatePeers(peers []PeerInfo)
}

// Logger interface provides structured logging capabilities
type Logger interface {
	Debug(msg string, fields ...Field)
	Info(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	Error(msg string, fields ...Field)
	Fatal(msg string, fields ...Field)
}

// Field represents a logging field
type Field struct {
	Key   string
	Value interface{}
}
