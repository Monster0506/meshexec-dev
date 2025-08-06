package internal

import (
	"time"
)

// MessageType represents the type of mesh message
type MessageType string

const (
	MessageTypeCommand MessageType = "cmd"
	MessageTypeResult  MessageType = "result"
	MessageTypePing    MessageType = "ping"
	MessageTypePong    MessageType = "pong"
)

// MeshMessage is the base message structure for all mesh communications
type MeshMessage struct {
	ID        string            `json:"id"`
	TTL       int              `json:"ttl"`
	Sender    string           `json:"sender"`
	Target    []string         `json:"target"`
	Type      MessageType      `json:"type"`
	Command   string           `json:"command,omitempty"`
	Payload   []byte           `json:"payload,omitempty"`
	Signature string           `json:"signature"`
	Timestamp int64            `json:"timestamp"`
}

// CommandMessage represents a command execution request
type CommandMessage struct {
	MeshMessage
	Command   string   `json:"command"`
	Arguments []string `json:"arguments,omitempty"`
	WorkDir   string   `json:"workdir,omitempty"`
	Timeout   int      `json:"timeout,omitempty"`
}

// ResultMessage represents a command execution result
type ResultMessage struct {
	MeshMessage
	CommandID string          `json:"command_id"`
	Result    ExecutionResult `json:"result"`
}

// ExecutionResult represents the result of a command execution
type ExecutionResult struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"code"`
	Device   string `json:"device"`
	Duration int64  `json:"duration_ms"`
}

// ExecutionResults represents aggregated results from multiple devices
type ExecutionResults struct {
	CommandID string            `json:"command_id"`
	Command   string            `json:"command"`
	Target    string            `json:"target"`
	Results   []ExecutionResult `json:"results"`
	Summary   ResultSummary     `json:"summary"`
	Timestamp time.Time         `json:"timestamp"`
}

// ResultSummary provides a summary of execution results
type ResultSummary struct {
	TotalDevices    int `json:"total_devices"`
	Successful      int `json:"successful"`
	Failed          int `json:"failed"`
	Timeout         int `json:"timeout"`
	AverageDuration int64 `json:"average_duration_ms"`
}

// DryRunResults represents the results of a dry-run operation
type DryRunResults struct {
	Command     string        `json:"command"`
	Target      string        `json:"target"`
	TargetedDevices []DeviceInfo `json:"targeted_devices"`
	WouldExecute bool          `json:"would_execute"`
	SafetyCheck  SafetyCheck   `json:"safety_check"`
}

// SafetyCheck represents the safety validation results
type SafetyCheck struct {
	IsSafe     bool     `json:"is_safe"`
	Warnings   []string `json:"warnings"`
	Blocked    bool     `json:"blocked"`
	BlockReason string  `json:"block_reason,omitempty"`
}

// PeerInfo represents information about a peer device in the mesh
type PeerInfo struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Address       string            `json:"address"`
	Role          string            `json:"role"`
	OS            string            `json:"os"`
	Arch          string            `json:"arch"`
	Tags          map[string]string `json:"tags"`
	LastSeen      time.Time         `json:"last_seen"`
	Connected     bool              `json:"connected"`
	SignalStrength int              `json:"signal_strength"`
}

// NetworkStatus represents the current status of the mesh network
type NetworkStatus struct {
	LocalNode     PeerInfo            `json:"local_node"`
	Peers         []PeerInfo          `json:"peers"`
	Routes        map[string][]string `json:"routes"`
	Updated       time.Time           `json:"updated"`
	TotalPeers    int                 `json:"total_peers"`
	ConnectedPeers int                `json:"connected_peers"`
}

// DeviceInfo represents device information for targeting
type DeviceInfo struct {
	Name string            `json:"name"`
	Role string            `json:"role"`
	OS   string            `json:"os"`
	Arch string            `json:"arch"`
	Tags map[string]string `json:"tags"`
}

// TargetAST represents the abstract syntax tree for target expressions
type TargetAST struct {
	Type     string      `json:"type"`
	Operator string      `json:"operator,omitempty"`
	Left     *TargetAST  `json:"left,omitempty"`
	Right    *TargetAST  `json:"right,omitempty"`
	Value    string      `json:"value,omitempty"`
}

// RunOptions represents options for command execution
type RunOptions struct {
	DryRun    bool   `json:"dry_run"`
	Timeout   int    `json:"timeout"`
	WorkDir   string `json:"workdir"`
	SafeMode  bool   `json:"safe_mode"`
}

// Advertisement represents a Bluetooth LE advertisement
type Advertisement struct {
	Address     string            `json:"address"`
	Name        string            `json:"name"`
	ServiceData map[string][]byte `json:"service_data"`
	RSSI        int               `json:"rssi"`
	Timestamp   time.Time         `json:"timestamp"`
}

// Connection represents a Bluetooth LE connection
type Connection struct {
	Address string `json:"address"`
	MTU     int    `json:"mtu"`
	Connected bool `json:"connected"`
}

// GATTService represents a GATT service
type GATTService struct {
	UUID         string            `json:"uuid"`
	Characteristics []GATTCharacteristic `json:"characteristics"`
}

// GATTCharacteristic represents a GATT characteristic
type GATTCharacteristic struct {
	UUID    string `json:"uuid"`
	Value   []byte `json:"value"`
	Writable bool  `json:"writable"`
}

// ErrorType represents the type of error
type ErrorType string

const (
	ErrorTypeNetwork   ErrorType = "network"
	ErrorTypeExecution ErrorType = "execution"
	ErrorTypeSecurity  ErrorType = "security"
	ErrorTypeConfig    ErrorType = "config"
	ErrorTypeTargeting ErrorType = "targeting"
)

// MechExecError represents a structured error
type MechExecError struct {
	Type    ErrorType                `json:"type"`
	Message string                   `json:"message"`
	Code    string                   `json:"code"`
	Details map[string]interface{}   `json:"details,omitempty"`
}

func (e *MechExecError) Error() string {
	return e.Message
} 