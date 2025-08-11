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
	handler := NewMessageHandlerWithLevel("none")
	assert.NotNil(t, handler)
}

func TestCreateCommandMessage(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

	msg := handler.CreatePingMessage("device1")

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "device1", msg.Sender)
	assert.Equal(t, internal.MessageTypePing, msg.Type)
	assert.Equal(t, 2, msg.TTL)
	assert.Empty(t, msg.Target)
}

func TestCreatePongMessage(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

	_, err := handler.DeserializeMessage([]byte("invalid json"))
	assert.Error(t, err)
}

func TestDeserializeMessage_UnknownType(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

	data := []byte(`{"type": "unknown"}`)
	_, err := handler.DeserializeMessage(data)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown message type")
}

func TestValidateMessage_Command(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

	msg := handler.CreatePingMessage("device1")

	err := handler.ValidateMessage(msg)
	assert.NoError(t, err)
}

func TestValidateMessage_Mesh_EmptyID(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

	msg := handler.CreatePingMessage("device1")
	msg.ID = ""

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message ID cannot be empty")
}

func TestValidateMessage_Mesh_EmptySender(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

	msg := handler.CreatePingMessage("")

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "sender cannot be empty")
}

func TestValidateMessage_Mesh_InvalidTTL(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

	msg := handler.CreatePingMessage("device1")
	msg.TTL = 0

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "TTL must be greater than 0")
}

func TestValidateMessage_Mesh_OldTimestamp(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

	msg := handler.CreatePingMessage("device1")
	msg.Timestamp = time.Now().Add(-2 * time.Hour).Unix() // 2 hours ago

	err := handler.ValidateMessage(msg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message is too old")
}

func TestValidateMessage_UnknownType(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

	err := handler.ValidateMessage("not a message")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown message type")
}

func TestDecrementTTL(t *testing.T) {
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

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
	handler := NewMessageHandlerWithLevel("none")

	id1 := handler.generateMessageID()
	id2 := handler.generateMessageID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEqual(t, id1, id2) // IDs should be unique

	// Should be valid UUID format
	assert.Len(t, id1, 36) // UUID v4 length
	assert.Len(t, id2, 36)
}

// Consolidated from zero-hit coverage tests
func TestMessageCreationAndSerialization_Basics(t *testing.T) {
	h := NewMessageHandler()

	cmd := h.CreateCommandMessage("echo", []string{"hello"}, []string{"all"}, "sender1", ".", 1000)
	if cmd == nil || cmd.ID == "" || cmd.Type != internal.MessageTypeCommand {
		t.Fatalf("unexpected command message: %+v", cmd)
	}

	res := h.CreateResultMessage(cmd.ID, internal.ExecutionResult{ID: "r1", Device: "d1"}, "sender1")
	if res == nil || res.ID == "" || res.Type != internal.MessageTypeResult {
		t.Fatalf("unexpected result message: %+v", res)
	}

	ping := h.CreatePingMessage("sender1")
	if ping == nil || ping.Type != internal.MessageTypePing {
		t.Fatalf("unexpected ping: %+v", ping)
	}

	pong := h.CreatePongMessage(ping.ID, "sender1")
	if pong == nil || pong.Type != internal.MessageTypePong {
		t.Fatalf("unexpected pong: %+v", pong)
	}

	data, err := h.SerializeMessage(cmd)
	if err != nil || len(data) == 0 {
		t.Fatalf("SerializeMessage failed: %v", err)
	}

	var tmp map[string]any
	if err := json.Unmarshal(data, &tmp); err != nil {
		t.Fatalf("serialized data not valid JSON: %v", err)
	}
}

func TestCreateExecutionResultAndResultsSummary_Basics(t *testing.T) {
	h := NewMessageHandler()

	r1 := h.CreateExecutionResult("echo", "success", "ok", "", 0, "dev1", 100*time.Millisecond)
	r2 := h.CreateExecutionResult("echo", "failed", "", "err", 1, "dev2", 200*time.Millisecond)
	r3 := h.CreateExecutionResult("echo", "timeout", "", "", 124, "dev3", 300*time.Millisecond)

	got := h.CreateExecutionResults("cmd-1", "echo hello", "all", []internal.ExecutionResult{r1, r2, r3})
	if got.CommandID != "cmd-1" || got.Command == "" || got.Target != "all" {
		t.Fatalf("unexpected aggregated results header: %+v", got)
	}
	if got.Summary.TotalDevices != 3 || got.Summary.Successful != 1 || got.Summary.Failed != 1 || got.Summary.Timeout != 1 {
		t.Fatalf("unexpected summary: %+v", got.Summary)
	}
	if got.Summary.AverageDuration == 0 {
		t.Fatalf("expected non-zero average duration: %+v", got.Summary)
	}
}

// Consolidated from messages_extra_test.go
func TestSerializeDeserialize_ErrorPaths(t *testing.T) {
	h := NewMessageHandlerWithLevel("none")
	// SerializeMessage error: channels/functions are unsupported by json.Marshal
	if _, err := h.SerializeMessage(make(chan int)); err == nil {
		t.Fatalf("expected serialize error for channel input")
	}

	// Deserialize invalid JSON
	if _, err := h.DeserializeMessage([]byte("{invalid json")); err == nil {
		t.Fatalf("expected deserialize error for invalid json")
	}

	// Unknown message type
	b := []byte(`{"type":"unknown"}`)
	if _, err := h.DeserializeMessage(b); err == nil {
		t.Fatalf("expected error for unknown message type")
	}
}

func TestValidateExecutionResult_Errors(t *testing.T) {
	h := NewMessageHandlerWithLevel("none")
	r := internal.ExecutionResult{}
	if err := h.ValidateMessage(&internal.ResultMessage{MeshMessage: internal.MeshMessage{ID: "x", TTL: 2, Sender: "s", Type: internal.MessageTypeResult, Timestamp: 1}, CommandID: "c", Result: r}); err == nil {
		t.Fatalf("expected error for empty result id/device")
	}

	// Negative duration
	r2 := internal.ExecutionResult{ID: "id", Device: "dev", Duration: -1}
	if err := h.ValidateMessage(&internal.ResultMessage{MeshMessage: internal.MeshMessage{ID: "x", TTL: 2, Sender: "s", Type: internal.MessageTypeResult, Timestamp: 1}, CommandID: "c", Result: r2}); err == nil {
		t.Fatalf("expected error for negative duration")
	}
}

func TestTTLAndExpiryHelpers(t *testing.T) {
	h := NewMessageHandlerWithLevel("none")
	cmd := &internal.CommandMessage{MeshMessage: internal.MeshMessage{ID: "1", TTL: 2, Sender: "s", Type: internal.MessageTypeCommand, Timestamp: 1}, Command: "echo"}
	if expired := h.IsExpired(cmd); expired {
		t.Fatalf("cmd should not be expired")
	}
	if ok := h.DecrementTTL(cmd); !ok {
		t.Fatalf("expected ttl remain > 0 after first decrement")
	}
	if ok := h.DecrementTTL(cmd); ok {
		t.Fatalf("expected ttl == 0 now")
	}
	if !h.IsExpired(cmd) {
		t.Fatalf("expected expired cmd")
	}

	res := &internal.ResultMessage{MeshMessage: internal.MeshMessage{ID: "2", TTL: 0, Sender: "s", Type: internal.MessageTypeResult, Timestamp: 1}, CommandID: "c", Result: internal.ExecutionResult{ID: "r", Device: "d"}}
	if !h.IsExpired(res) {
		t.Fatalf("result with ttl 0 should be expired")
	}

	base := &internal.MeshMessage{ID: "3", TTL: -1, Sender: "s", Type: internal.MessageTypePing, Timestamp: 1}
	if !h.IsExpired(base) {
		t.Fatalf("mesh with ttl -1 should be expired")
	}
}
