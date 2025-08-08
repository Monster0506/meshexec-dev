package messages

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessageHandler(t *testing.T) {
	handler := NewMessageHandler()
	assert.NotNil(t, handler)
}

func TestCreateCommandMessage(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1", "device2"},
		"sender1",
		"/tmp",
		30,
	)

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "ls", msg.Command)
	assert.Equal(t, []string{"-la"}, msg.Arguments)
	assert.Equal(t, []string{"device1", "device2"}, msg.Target)
	assert.Equal(t, "sender1", msg.Sender)
	assert.Equal(t, "/tmp", msg.WorkDir)
	assert.Equal(t, 30, msg.Timeout)
	assert.Equal(t, internal.MessageTypeCommand, msg.Type)
	assert.Equal(t, 5, msg.TTL)
	assert.Greater(t, msg.Timestamp, int64(0))
}

func TestCreateResultMessage(t *testing.T) {
	handler := NewMessageHandler()

	result := internal.ExecutionResult{
		ID:       "result1",
		Type:     "command_execution",
		Status:   "success",
		Stdout:   "file1.txt\nfile2.txt",
		Stderr:   "",
		ExitCode: 0,
		Device:   "device1",
		Duration: 100,
	}

	msg := handler.CreateResultMessage("cmd1", result, "device1")

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "cmd1", msg.CommandID)
	assert.Equal(t, result, msg.Result)
	assert.Equal(t, "device1", msg.Sender)
	assert.Equal(t, internal.MessageTypeResult, msg.Type)
	assert.Equal(t, 3, msg.TTL)
	assert.Empty(t, msg.Target) // Results are sent back to sender
}

func TestCreatePingMessage(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreatePingMessage("device1")

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "device1", msg.Sender)
	assert.Equal(t, internal.MessageTypePing, msg.Type)
	assert.Equal(t, 2, msg.TTL)
	assert.Empty(t, msg.Target)
}

func TestCreatePongMessage(t *testing.T) {
	handler := NewMessageHandler()

	pingID := "ping123"
	msg := handler.CreatePongMessage(pingID, "device1")

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "device1", msg.Sender)
	assert.Equal(t, internal.MessageTypePong, msg.Type)
	assert.Equal(t, 2, msg.TTL)
	assert.Equal(t, []byte(pingID), msg.Payload)
}

func TestSerializeMessage(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"echo",
		[]string{"hello"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		10,
	)

	data, err := handler.SerializeMessage(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	// Verify it's valid JSON
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "cmd", parsed["type"])
	assert.Equal(t, "echo", parsed["command"])
	assert.Equal(t, "sender1", parsed["sender"])
}

func TestDeserializeMessage_Command(t *testing.T) {
	handler := NewMessageHandler()

	originalMsg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	data, err := handler.SerializeMessage(originalMsg)
	require.NoError(t, err)

	deserialized, err := handler.DeserializeMessage(data)
	require.NoError(t, err)

	msg, ok := deserialized.(*internal.CommandMessage)
	require.True(t, ok)

	assert.Equal(t, originalMsg.ID, msg.ID)
	assert.Equal(t, originalMsg.Command, msg.Command)
	assert.Equal(t, originalMsg.Arguments, msg.Arguments)
	assert.Equal(t, originalMsg.Target, msg.Target)
	assert.Equal(t, originalMsg.Sender, msg.Sender)
	assert.Equal(t, originalMsg.Type, msg.Type)
}

func TestDeserializeMessage_Result(t *testing.T) {
	handler := NewMessageHandler()

	result := internal.ExecutionResult{
		ID:       "result1",
		Type:     "command_execution",
		Status:   "success",
		Stdout:   "output",
		Stderr:   "",
		ExitCode: 0,
		Device:   "device1",
		Duration: 100,
	}

	originalMsg := handler.CreateResultMessage("cmd1", result, "device1")

	data, err := handler.SerializeMessage(originalMsg)
	require.NoError(t, err)

	deserialized, err := handler.DeserializeMessage(data)
	require.NoError(t, err)

	msg, ok := deserialized.(*internal.ResultMessage)
	require.True(t, ok)

	assert.Equal(t, originalMsg.ID, msg.ID)
	assert.Equal(t, originalMsg.CommandID, msg.CommandID)
	assert.Equal(t, originalMsg.Result, msg.Result)
	assert.Equal(t, originalMsg.Type, msg.Type)
}

func TestDeserializeMessage_Ping(t *testing.T) {
	handler := NewMessageHandler()

	originalMsg := handler.CreatePingMessage("device1")

	data, err := handler.SerializeMessage(originalMsg)
	require.NoError(t, err)

	deserialized, err := handler.DeserializeMessage(data)
	require.NoError(t, err)

	msg, ok := deserialized.(*internal.MeshMessage)
	require.True(t, ok)

	assert.Equal(t, originalMsg.ID, msg.ID)
	assert.Equal(t, originalMsg.Sender, msg.Sender)
	assert.Equal(t, originalMsg.Type, msg.Type)
}

func TestDeserializeMessage_InvalidJSON(t *testing.T) {
	handler := NewMessageHandler()

	_, err := handler.DeserializeMessage([]byte("invalid json"))
	assert.Error(t, err)
}

func TestDeserializeMessage_UnknownType(t *testing.T) {
	handler := NewMessageHandler()

	data := []byte(`{"type": "unknown"}`)
	_, err := handler.DeserializeMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown message type")
}

func TestValidateMessage_Command(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	err := handler.ValidateMessage(msg)
	assert.NoError(t, err)
}

func TestValidateMessage_Command_EmptyCommand(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command cannot be empty")
}

func TestValidateMessage_Command_InvalidType(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)
	msg.Type = internal.MessageTypeResult // Wrong type

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid message type for command message")
}

func TestValidateMessage_Result(t *testing.T) {
	handler := NewMessageHandler()

	result := internal.ExecutionResult{
		ID:       "result1",
		Type:     "command_execution",
		Status:   "success",
		Stdout:   "output",
		Stderr:   "",
		ExitCode: 0,
		Device:   "device1",
		Duration: 100,
	}

	msg := handler.CreateResultMessage("cmd1", result, "device1")

	err := handler.ValidateMessage(msg)
	assert.NoError(t, err)
}

func TestValidateMessage_Result_EmptyCommandID(t *testing.T) {
	handler := NewMessageHandler()

	result := internal.ExecutionResult{
		ID:       "result1",
		Type:     "command_execution",
		Status:   "success",
		Stdout:   "output",
		Stderr:   "",
		ExitCode: 0,
		Device:   "device1",
		Duration: 100,
	}

	msg := handler.CreateResultMessage("", result, "device1")

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "command ID cannot be empty")
}

func TestValidateMessage_Mesh(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreatePingMessage("device1")

	err := handler.ValidateMessage(msg)
	assert.NoError(t, err)
}

func TestValidateMessage_Mesh_EmptyID(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreatePingMessage("device1")
	msg.ID = ""

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")
}

func TestValidateMessage_Mesh_EmptySender(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreatePingMessage("")

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sender cannot be empty")
}

func TestValidateMessage_Mesh_InvalidTTL(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreatePingMessage("device1")
	msg.TTL = 0

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL must be greater than 0")
}

func TestValidateMessage_Mesh_OldTimestamp(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreatePingMessage("device1")
	msg.Timestamp = time.Now().Add(-2 * time.Hour).Unix() // 2 hours ago

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is too old")
}

func TestValidateMessage_UnknownType(t *testing.T) {
	handler := NewMessageHandler()

	err := handler.ValidateMessage("not a message")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown message type")
}

func TestDecrementTTL(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	originalTTL := msg.TTL

	// Decrement TTL
	stillValid := handler.DecrementTTL(msg)
	assert.True(t, stillValid)
	assert.Equal(t, originalTTL-1, msg.TTL)

	// Decrement until expired (TTL starts at 5, so we need 3 more decrements to get to 1)
	for i := 0; i < 3; i++ {
		stillValid = handler.DecrementTTL(msg)
		assert.True(t, stillValid)
	}

	// Final decrement should make it expired
	stillValid = handler.DecrementTTL(msg)
	assert.False(t, stillValid)
	assert.Equal(t, 0, msg.TTL)
}

func TestIsExpired(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	// Should not be expired initially
	assert.False(t, handler.IsExpired(msg))

	// Set TTL to 0
	msg.TTL = 0
	assert.True(t, handler.IsExpired(msg))

	// Set TTL to negative
	msg.TTL = -1
	assert.True(t, handler.IsExpired(msg))
}

func TestGetMessageID(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	id := handler.GetMessageID(msg)
	assert.Equal(t, msg.ID, id)
	assert.NotEmpty(t, id)
}

func TestGetMessageType(t *testing.T) {
	handler := NewMessageHandler()

	msg := handler.CreateCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1"},
		"sender1",
		"/tmp",
		30,
	)

	msgType := handler.GetMessageType(msg)
	assert.Equal(t, internal.MessageTypeCommand, msgType)
}

func TestCreateExecutionResult(t *testing.T) {
	handler := NewMessageHandler()

	result := handler.CreateExecutionResult(
		"ls -la",
		"success",
		"file1.txt\nfile2.txt",
		"",
		0,
		"device1",
		100*time.Millisecond,
	)

	assert.NotEmpty(t, result.ID)
	assert.Equal(t, "command_execution", result.Type)
	assert.Equal(t, "success", result.Status)
	assert.Equal(t, "file1.txt\nfile2.txt", result.Stdout)
	assert.Equal(t, "", result.Stderr)
	assert.Equal(t, 0, result.ExitCode)
	assert.Equal(t, "device1", result.Device)
	assert.Equal(t, int64(100), result.Duration)
}

func TestCreateExecutionResults(t *testing.T) {
	handler := NewMessageHandler()

	results := []internal.ExecutionResult{
		{
			ID:       "result1",
			Type:     "command_execution",
			Status:   "success",
			Stdout:   "output1",
			Stderr:   "",
			ExitCode: 0,
			Device:   "device1",
			Duration: 100,
		},
		{
			ID:       "result2",
			Type:     "command_execution",
			Status:   "failed",
			Stdout:   "",
			Stderr:   "error",
			ExitCode: 1,
			Device:   "device2",
			Duration: 200,
		},
	}

	execResults := handler.CreateExecutionResults("cmd1", "ls -la", "all", results)

	assert.Equal(t, "cmd1", execResults.CommandID)
	assert.Equal(t, "ls -la", execResults.Command)
	assert.Equal(t, "all", execResults.Target)
	assert.Equal(t, results, execResults.Results)
	assert.Equal(t, 2, execResults.Summary.TotalDevices)
	assert.Equal(t, 1, execResults.Summary.Successful)
	assert.Equal(t, 1, execResults.Summary.Failed)
	assert.Equal(t, 0, execResults.Summary.Timeout)
	assert.Equal(t, int64(150), execResults.Summary.AverageDuration)
}

func TestCalculateResultSummary(t *testing.T) {
	handler := NewMessageHandler()

	results := []internal.ExecutionResult{
		{Status: "success", Duration: 100},
		{Status: "success", Duration: 200},
		{Status: "failed", Duration: 150},
		{Status: "timeout", Duration: 300},
	}

	summary := handler.calculateResultSummary(results)

	assert.Equal(t, 4, summary.TotalDevices)
	assert.Equal(t, 2, summary.Successful)
	assert.Equal(t, 1, summary.Failed)
	assert.Equal(t, 1, summary.Timeout)
	assert.Equal(t, int64(187), summary.AverageDuration) // (100+200+150+300)/4 = 187.5, truncated to 187
}

func TestGenerateMessageID(t *testing.T) {
	handler := NewMessageHandler()

	id1 := handler.generateMessageID()
	id2 := handler.generateMessageID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // IDs should be unique

	// Should be valid UUID format
	assert.Len(t, id1, 36) // UUID v4 length
	assert.Len(t, id2, 36)
}
