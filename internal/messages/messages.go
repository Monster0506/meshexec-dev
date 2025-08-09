package messages

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// MessageHandler provides utilities for creating, serializing, and validating messages
type MessageHandler struct {
	logger *logging.Logger
}

// NewMessageHandler creates a new message handler
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		logger: logging.NewLogger("info"),
	}
}

// NewMessageHandlerWithLevel creates a new message handler with a configurable log level.
// Use level "none" in tests to silence logs.
func NewMessageHandlerWithLevel(level string) *MessageHandler {
	return &MessageHandler{
		logger: logging.NewLogger(level),
	}
}

// CreateCommandMessage creates a new command message
func (h *MessageHandler) CreateCommandMessage(
	command string,
	arguments []string,
	target []string,
	sender string,
	workDir string,
	timeout int,
) *internal.CommandMessage {
	msg := &internal.CommandMessage{
		MeshMessage: internal.MeshMessage{
			ID:        h.generateMessageID(),
			TTL:       5, // Default TTL
			Sender:    sender,
			Target:    target,
			Type:      internal.MessageTypeCommand,
			Timestamp: time.Now().Unix(),
		},
		Command:   command,
		Arguments: arguments,
		WorkDir:   workDir,
		Timeout:   timeout,
	}

	h.logger.Debug("Created command message", map[string]interface{}{
		"message_id": msg.ID,
		"command":    command,
		"sender":     sender,
		"targets":    target,
		"timeout":    timeout,
	})

	return msg
}

// CreateResultMessage creates a new result message
func (h *MessageHandler) CreateResultMessage(
	commandID string,
	result internal.ExecutionResult,
	sender string,
) *internal.ResultMessage {
	msg := &internal.ResultMessage{
		MeshMessage: internal.MeshMessage{
			ID:        h.generateMessageID(),
			TTL:       3, // Lower TTL for results
			Sender:    sender,
			Target:    []string{}, // Results are sent back to sender
			Type:      internal.MessageTypeResult,
			Timestamp: time.Now().Unix(),
		},
		CommandID: commandID,
		Result:    result,
	}
	return msg
}

// CreatePingMessage creates a new ping message
func (h *MessageHandler) CreatePingMessage(sender string) *internal.MeshMessage {
	return &internal.MeshMessage{
		ID:        h.generateMessageID(),
		TTL:       2,
		Sender:    sender,
		Target:    []string{},
		Type:      internal.MessageTypePing,
		Timestamp: time.Now().Unix(),
	}
}

// CreatePongMessage creates a new pong message in response to a ping
func (h *MessageHandler) CreatePongMessage(pingID string, sender string) *internal.MeshMessage {
	return &internal.MeshMessage{
		ID:        h.generateMessageID(),
		TTL:       2,
		Sender:    sender,
		Target:    []string{},
		Type:      internal.MessageTypePong,
		Payload:   []byte(pingID), // Include original ping ID
		Timestamp: time.Now().Unix(),
	}
}

// SerializeMessage serializes a message to JSON
func (h *MessageHandler) SerializeMessage(msg interface{}) ([]byte, error) {
	h.logger.Debug("Serializing message", map[string]interface{}{
		"message_type": fmt.Sprintf("%T", msg),
	})

	data, err := json.Marshal(msg)
	if err != nil {
		h.logger.Error("Failed to serialize message", err, map[string]interface{}{
			"message_type": fmt.Sprintf("%T", msg),
		})
		return nil, fmt.Errorf("failed to serialize message: %w", err)
	}

	h.logger.Debug("Message serialized successfully", map[string]interface{}{
		"message_type": fmt.Sprintf("%T", msg),
		"data_length":  len(data),
	})

	return data, nil
}

// DeserializeMessage deserializes a JSON message to the appropriate type
func (h *MessageHandler) DeserializeMessage(data []byte) (interface{}, error) {
	h.logger.Debug("Deserializing message", map[string]interface{}{
		"data_length": len(data),
	})

	// First, try to determine the message type
	var baseMsg struct {
		Type internal.MessageType `json:"type"`
	}

	if err := json.Unmarshal(data, &baseMsg); err != nil {
		h.logger.Error("Failed to determine message type", err, map[string]interface{}{
			"data_length": len(data),
		})
		return nil, fmt.Errorf("failed to determine message type: %w", err)
	}

	h.logger.Debug("Determined message type", map[string]interface{}{
		"message_type": baseMsg.Type,
		"data_length":  len(data),
	})

	// Deserialize based on message type
	switch baseMsg.Type {
	case internal.MessageTypeCommand:
		var msg internal.CommandMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.logger.Error("Failed to deserialize command message", err, map[string]interface{}{
				"message_type": baseMsg.Type,
			})
			return nil, fmt.Errorf("failed to deserialize command message: %w", err)
		}
		h.logger.Debug("Deserialized command message", map[string]interface{}{
			"message_id": msg.ID,
			"command":    msg.Command,
		})
		return &msg, nil

	case internal.MessageTypeResult:
		var msg internal.ResultMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.logger.Error("Failed to deserialize result message", err, map[string]interface{}{
				"message_type": baseMsg.Type,
			})
			return nil, fmt.Errorf("failed to deserialize result message: %w", err)
		}
		h.logger.Debug("Deserialized result message", map[string]interface{}{
			"message_id": msg.ID,
			"command_id": msg.CommandID,
		})
		return &msg, nil

	case internal.MessageTypePing, internal.MessageTypePong:
		var msg internal.MeshMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			h.logger.Error("Failed to deserialize mesh message", err, map[string]interface{}{
				"message_type": baseMsg.Type,
			})
			return nil, fmt.Errorf("failed to deserialize mesh message: %w", err)
		}
		h.logger.Debug("Deserialized mesh message", map[string]interface{}{
			"message_id":   msg.ID,
			"message_type": msg.Type,
		})
		return &msg, nil

	default:
		h.logger.Error("Unknown message type", fmt.Errorf("unknown message type: %s", baseMsg.Type), map[string]interface{}{
			"message_type": baseMsg.Type,
		})
		return nil, fmt.Errorf("unknown message type: %s", baseMsg.Type)
	}
}

// ValidateMessage validates a message structure
func (h *MessageHandler) ValidateMessage(msg interface{}) error {
	switch m := msg.(type) {
	case *internal.CommandMessage:
		return h.validateCommandMessage(m)
	case *internal.ResultMessage:
		return h.validateResultMessage(m)
	case *internal.MeshMessage:
		return h.validateMeshMessage(m)
	default:
		return fmt.Errorf("unknown message type: %T", msg)
	}
}

// validateCommandMessage validates a command message
func (h *MessageHandler) validateCommandMessage(msg *internal.CommandMessage) error {
	if err := h.validateMeshMessage(&msg.MeshMessage); err != nil {
		return err
	}

	if msg.Command == "" {
		return fmt.Errorf("command cannot be empty")
	}

	if msg.Type != internal.MessageTypeCommand {
		return fmt.Errorf("invalid message type for command message: %s", msg.Type)
	}

	if msg.TTL <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}

	return nil
}

// validateResultMessage validates a result message
func (h *MessageHandler) validateResultMessage(msg *internal.ResultMessage) error {
	if err := h.validateMeshMessage(&msg.MeshMessage); err != nil {
		return err
	}

	if msg.CommandID == "" {
		return fmt.Errorf("command ID cannot be empty")
	}

	if msg.Type != internal.MessageTypeResult {
		return fmt.Errorf("invalid message type for result message: %s", msg.Type)
	}

	if err := h.validateExecutionResult(&msg.Result); err != nil {
		return fmt.Errorf("invalid execution result: %w", err)
	}

	return nil
}

// validateMeshMessage validates a base mesh message
func (h *MessageHandler) validateMeshMessage(msg *internal.MeshMessage) error {
	if msg.ID == "" {
		return fmt.Errorf("message ID cannot be empty")
	}

	if msg.Sender == "" {
		return fmt.Errorf("sender cannot be empty")
	}

	if msg.TTL <= 0 {
		return fmt.Errorf("TTL must be greater than 0")
	}

	if msg.Timestamp <= 0 {
		return fmt.Errorf("timestamp must be greater than 0")
	}

	// Check if message is too old (more than 1 hour)
	if time.Now().Unix()-msg.Timestamp > 3600 {
		return fmt.Errorf("message is too old")
	}

	return nil
}

// validateExecutionResult validates an execution result
func (h *MessageHandler) validateExecutionResult(result *internal.ExecutionResult) error {
	if result.ID == "" {
		return fmt.Errorf("result ID cannot be empty")
	}

	if result.Device == "" {
		return fmt.Errorf("device cannot be empty")
	}

	if result.Duration < 0 {
		return fmt.Errorf("duration cannot be negative")
	}

	return nil
}

// DecrementTTL decrements the TTL of a message
func (h *MessageHandler) DecrementTTL(msg interface{}) bool {
	switch m := msg.(type) {
	case *internal.CommandMessage:
		if m.TTL > 0 {
			m.TTL--
		}
		return m.TTL > 0
	case *internal.ResultMessage:
		if m.TTL > 0 {
			m.TTL--
		}
		return m.TTL > 0
	case *internal.MeshMessage:
		if m.TTL > 0 {
			m.TTL--
		}
		return m.TTL > 0
	default:
		return false
	}
}

// IsExpired checks if a message has expired (TTL <= 0)
func (h *MessageHandler) IsExpired(msg interface{}) bool {
	switch m := msg.(type) {
	case *internal.CommandMessage:
		return m.TTL <= 0
	case *internal.ResultMessage:
		return m.TTL <= 0
	case *internal.MeshMessage:
		return m.TTL <= 0
	default:
		return true
	}
}

// GetMessageID returns the ID of a message
func (h *MessageHandler) GetMessageID(msg interface{}) string {
	switch m := msg.(type) {
	case *internal.CommandMessage:
		return m.ID
	case *internal.ResultMessage:
		return m.ID
	case *internal.MeshMessage:
		return m.ID
	default:
		return ""
	}
}

// GetMessageType returns the type of a message
func (h *MessageHandler) GetMessageType(msg interface{}) internal.MessageType {
	switch m := msg.(type) {
	case *internal.CommandMessage:
		return m.Type
	case *internal.ResultMessage:
		return m.Type
	case *internal.MeshMessage:
		return m.Type
	default:
		return ""
	}
}

// generateMessageID generates a unique message ID
func (h *MessageHandler) generateMessageID() string {
	// Use UUID v4 for message IDs
	id := uuid.New()
	return id.String()
}

// CreateExecutionResult creates a new execution result
func (h *MessageHandler) CreateExecutionResult(
	command string,
	status string,
	stdout string,
	stderr string,
	exitCode int,
	device string,
	duration time.Duration,
) internal.ExecutionResult {
	return internal.ExecutionResult{
		ID:       h.generateMessageID(),
		Type:     "command_execution",
		Status:   status,
		Stdout:   stdout,
		Stderr:   stderr,
		ExitCode: exitCode,
		Device:   device,
		Duration: duration.Milliseconds(),
	}
}

// CreateExecutionResults creates aggregated execution results
func (h *MessageHandler) CreateExecutionResults(
	commandID string,
	command string,
	target string,
	results []internal.ExecutionResult,
) internal.ExecutionResults {
	summary := h.calculateResultSummary(results)

	return internal.ExecutionResults{
		CommandID: commandID,
		Command:   command,
		Target:    target,
		Results:   results,
		Summary:   summary,
		Timestamp: time.Now(),
	}
}

// calculateResultSummary calculates summary statistics from execution results
func (h *MessageHandler) calculateResultSummary(results []internal.ExecutionResult) internal.ResultSummary {
	summary := internal.ResultSummary{
		TotalDevices: len(results),
	}

	var totalDuration int64
	for _, result := range results {
		switch result.Status {
		case "success":
			summary.Successful++
		case "failed":
			summary.Failed++
		case "timeout":
			summary.Timeout++
		}
		totalDuration += result.Duration
	}

	if len(results) > 0 {
		summary.AverageDuration = totalDuration / int64(len(results))
	}

	return summary
}
