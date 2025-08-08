package security

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityManager(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")
	assert.NotNil(t, sm)
	assert.False(t, sm.HasKeys())
}

func TestGenerateKeyPair(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	assert.True(t, sm.HasKeys())
	assert.NotNil(t, sm.GetPublicKeyBytes())
	assert.NotEmpty(t, sm.GetPublicKey())

	// Verify key sizes
	publicKey := sm.GetPublicKeyBytes()
	assert.Len(t, publicKey, 32) // ed25519 public key size
}

func TestSaveAndLoadKeys(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "test_keys")

    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	// Save keys
	err = sm.SaveKeys(keyPath)
	require.NoError(t, err)

	// Verify files were created
	privateKeyPath := keyPath + ".private"
	publicKeyPath := keyPath + ".public"

	assert.FileExists(t, privateKeyPath)
	assert.FileExists(t, publicKeyPath)

	// Check file permissions (skip on Windows due to different permission handling)
	if os.PathSeparator != '\\' { // Not Windows
		privateKeyInfo, err := os.Stat(privateKeyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0600), privateKeyInfo.Mode().Perm())

		publicKeyInfo, err := os.Stat(publicKeyPath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0644), publicKeyInfo.Mode().Perm())
	}

	// Create new security manager and load keys
    sm2 := NewSecurityManagerWithLevel("none")
	err = sm2.LoadKeys(keyPath)
	require.NoError(t, err)

	assert.True(t, sm2.HasKeys())
	assert.Equal(t, sm.GetPublicKey(), sm2.GetPublicKey())
}

func TestLoadKeys_GenerateNewIfNotExist(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "nonexistent_keys")

    sm := NewSecurityManagerWithLevel("none")

	// Load keys (should generate new ones)
	err := sm.LoadKeys(keyPath)
	require.NoError(t, err)

	assert.True(t, sm.HasKeys())
	assert.NotEmpty(t, sm.GetPublicKey())

	// Verify files were created
	privateKeyPath := keyPath + ".private"
	publicKeyPath := keyPath + ".public"

	assert.FileExists(t, privateKeyPath)
	assert.FileExists(t, publicKeyPath)
}

func TestSignAndVerifyMessage(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	// Create a test message
	testMsg := map[string]interface{}{
		"command":   "ls",
		"target":    []string{"device1"},
		"timestamp": time.Now().Unix(),
	}

	// Sign the message
	signature, err := sm.SignMessage(testMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify the signature
	err = sm.VerifyMessage(testMsg, signature)
	assert.NoError(t, err)

	// Verify with wrong signature
	err = sm.VerifyMessage(testMsg, "invalid_signature")
	assert.Error(t, err)
	// The error could be either "failed to decode signature" or "signature verification failed"
	assert.True(t,
		strings.Contains(err.Error(), "signature verification failed") ||
			strings.Contains(err.Error(), "failed to decode signature"),
		"Expected error about signature verification or decoding, got: %s", err.Error())

	// Verify with modified message
	modifiedMsg := map[string]interface{}{
		"command":   "ls",
		"target":    []string{"device2"}, // Different target
		"timestamp": time.Now().Unix(),
	}
	err = sm.VerifyMessage(modifiedMsg, signature)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestSignAndVerifyMeshMessage(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	// Create a mesh message
	msg := &internal.MeshMessage{
		ID:        "test-msg-1",
		TTL:       5,
		Sender:    "device1",
		Target:    []string{"device2"},
		Type:      internal.MessageTypeCommand,
		Timestamp: time.Now().Unix(),
	}

	// Sign the message
	err = sm.SignMeshMessage(msg)
	require.NoError(t, err)
	assert.NotEmpty(t, msg.Signature)

	// Verify the signature
	err = sm.VerifyMeshMessage(msg)
	assert.NoError(t, err)

	// Verify with empty signature
	msg2 := &internal.MeshMessage{
		ID:        "test-msg-2",
		TTL:       5,
		Sender:    "device1",
		Target:    []string{"device2"},
		Type:      internal.MessageTypeCommand,
		Timestamp: time.Now().Unix(),
	}
	err = sm.VerifyMeshMessage(msg2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "message has no signature")
}

func TestCreateSignedCommandMessage(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	// Create signed command message
	msg, err := sm.CreateSignedCommandMessage(
		"ls",
		[]string{"-la"},
		[]string{"device1", "device2"},
		"device0",
		"/tmp",
		30,
	)
	require.NoError(t, err)

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "ls", msg.Command)
	assert.Equal(t, []string{"-la"}, msg.Arguments)
	assert.Equal(t, []string{"device1", "device2"}, msg.Target)
	assert.Equal(t, "device0", msg.Sender)
	assert.Equal(t, "/tmp", msg.WorkDir)
	assert.Equal(t, 30, msg.Timeout)
	assert.Equal(t, internal.MessageTypeCommand, msg.Type)
	assert.Equal(t, 5, msg.TTL)
	assert.NotEmpty(t, msg.Signature)

	// Verify the signature
	err = sm.VerifyMeshMessage(&msg.MeshMessage)
	assert.NoError(t, err)
}

func TestCreateSignedResultMessage(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	// Create execution result
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

	// Create signed result message
	msg, err := sm.CreateSignedResultMessage("cmd1", result, "device1")
	require.NoError(t, err)

	assert.NotNil(t, msg)
	assert.NotEmpty(t, msg.ID)
	assert.Equal(t, "cmd1", msg.CommandID)
	assert.Equal(t, result, msg.Result)
	assert.Equal(t, "device1", msg.Sender)
	assert.Equal(t, internal.MessageTypeResult, msg.Type)
	assert.Equal(t, 3, msg.TTL)
	assert.Empty(t, msg.Target)
	assert.NotEmpty(t, msg.Signature)

	// Verify the signature
	err = sm.VerifyMeshMessage(&msg.MeshMessage)
	assert.NoError(t, err)
}

func TestSignMessage_NoPrivateKey(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	testMsg := map[string]string{"test": "data"}

	_, err := sm.SignMessage(testMsg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "private key not loaded")
}

func TestVerifyMessage_NoPublicKey(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	testMsg := map[string]string{"test": "data"}

	err := sm.VerifyMessage(testMsg, "signature")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "public key not loaded")
}

func TestSaveKeys_NoKeys(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	err := sm.SaveKeys("/tmp/test_keys")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no keys to save")
}

func TestLoadKeys_InvalidKeyData(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "invalid_keys")

	// Create invalid key files
	privateKeyPath := keyPath + ".private"
	publicKeyPath := keyPath + ".public"

	// Write invalid base64 data
	err := os.WriteFile(privateKeyPath, []byte("invalid_base64"), 0600)
	require.NoError(t, err)

	err = os.WriteFile(publicKeyPath, []byte("invalid_base64"), 0644)
	require.NoError(t, err)

    sm := NewSecurityManagerWithLevel("none")

	err = sm.LoadKeys(keyPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode private key")
}

func TestLoadKeys_InvalidKeySize(t *testing.T) {
	tempDir := t.TempDir()
	keyPath := filepath.Join(tempDir, "invalid_size_keys")

	// Create key files with wrong sizes
	privateKeyPath := keyPath + ".private"
	publicKeyPath := keyPath + ".public"

	// Write valid base64 but wrong size data
	shortKey := "dGVzdA==" // "test" in base64
	err := os.WriteFile(privateKeyPath, []byte(shortKey), 0600)
	require.NoError(t, err)

	err = os.WriteFile(publicKeyPath, []byte(shortKey), 0644)
	require.NoError(t, err)

    sm := NewSecurityManagerWithLevel("none")

	err = sm.LoadKeys(keyPath)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid private key size")
}

func TestVerifyMessage_InvalidSignature(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	testMsg := map[string]string{"test": "data"}

	// Try to verify with invalid base64 signature
	err = sm.VerifyMessage(testMsg, "invalid_base64_signature")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to decode signature")
}

func TestMessageIDGeneration(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate multiple IDs and ensure they're unique
	id1 := sm.generateMessageID()
	id2 := sm.generateMessageID()
	id3 := sm.generateMessageID()

	assert.NotEmpty(t, id1)
	assert.NotEmpty(t, id2)
	assert.NotEmpty(t, id3)

	assert.NotEqual(t, id1, id2)
	assert.NotEqual(t, id1, id3)
	assert.NotEqual(t, id2, id3)

	// Verify format (timestamp-hex)
	assert.Contains(t, id1, "-")
}

func TestSignMessage_ComplexData(t *testing.T) {
    sm := NewSecurityManagerWithLevel("none")

	// Generate keys
	err := sm.GenerateKeyPair()
	require.NoError(t, err)

	// Create complex nested data structure
	complexMsg := map[string]interface{}{
		"command":   "ls",
		"arguments": []string{"-la", "-h"},
		"target": map[string]interface{}{
			"devices": []string{"device1", "device2"},
			"filters": map[string]string{
				"os":   "linux",
				"arch": "amd64",
			},
		},
		"metadata": map[string]interface{}{
			"timestamp": time.Now().Unix(),
			"user":      "admin",
			"session":   "abc123",
		},
	}

	// Sign the message
	signature, err := sm.SignMessage(complexMsg)
	require.NoError(t, err)
	assert.NotEmpty(t, signature)

	// Verify the signature
	err = sm.VerifyMessage(complexMsg, signature)
	assert.NoError(t, err)

	// Modify the message and verify it fails
	complexMsg["command"] = "rm"
	err = sm.VerifyMessage(complexMsg, signature)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}

func TestCrossKeyVerification(t *testing.T) {
	// Create two security managers with different keys
    sm1 := NewSecurityManagerWithLevel("none")
    sm2 := NewSecurityManagerWithLevel("none")

	// Generate keys for both
	err := sm1.GenerateKeyPair()
	require.NoError(t, err)

	err = sm2.GenerateKeyPair()
	require.NoError(t, err)

	// Ensure they have different keys
	assert.NotEqual(t, sm1.GetPublicKey(), sm2.GetPublicKey())

	// Create a message with sm1
	testMsg := map[string]string{"test": "data"}
	signature, err := sm1.SignMessage(testMsg)
	require.NoError(t, err)

	// Verify with sm1 (should work)
	err = sm1.VerifyMessage(testMsg, signature)
	assert.NoError(t, err)

	// Verify with sm2 (should fail)
	err = sm2.VerifyMessage(testMsg, signature)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "signature verification failed")
}
