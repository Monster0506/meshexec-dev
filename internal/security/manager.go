package security

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// SecurityManager handles cryptographic operations for message signing and verification
type SecurityManager struct {
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey
	keyPath    string
	logger     *logging.Logger
}

// NewSecurityManager creates a new security manager with default info log level
func NewSecurityManager() *SecurityManager {
	return &SecurityManager{
		logger: logging.NewLogger("info"),
	}
}

// NewSecurityManagerWithLevel allows tests and callers to control log level
func NewSecurityManagerWithLevel(level string) *SecurityManager {
	return &SecurityManager{
		logger: logging.NewLogger(level),
	}
}

// GenerateKeyPair generates a new ed25519 key pair
func (sm *SecurityManager) GenerateKeyPair() error {
	sm.logger.Debug("Generating new ed25519 key pair", nil)

	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		sm.logger.Error("Failed to generate key pair", err, nil)
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	sm.publicKey = publicKey
	sm.privateKey = privateKey

	sm.logger.Info("Key pair generated successfully", map[string]interface{}{
		"public_key_length":  len(publicKey),
		"private_key_length": len(privateKey),
	})

	return nil
}

// LoadKeys loads keys from the specified path
func (sm *SecurityManager) LoadKeys(keyPath string) error {
	sm.keyPath = keyPath
	sm.logger.Debug("Loading keys from path", map[string]interface{}{
		"key_path": keyPath,
	})

	// Try to load existing keys
	if err := sm.loadExistingKeys(); err != nil {
		// If keys don't exist, generate new ones
		if os.IsNotExist(err) {
			sm.logger.Info("Keys not found, generating new key pair", map[string]interface{}{
				"key_path": keyPath,
			})
			return sm.generateAndSaveKeys()
		}
		sm.logger.Error("Failed to load existing keys", err, map[string]interface{}{
			"key_path": keyPath,
		})
		return err
	}

	sm.logger.Info("Keys loaded successfully", map[string]interface{}{
		"key_path": keyPath,
	})
	return nil
}

// SaveKeys saves the current keys to the specified path
func (sm *SecurityManager) SaveKeys(keyPath string) error {
	sm.logger.Debug("Saving keys to path", map[string]interface{}{
		"key_path": keyPath,
	})

	if sm.privateKey == nil || sm.publicKey == nil {
		sm.logger.Error("No keys to save", fmt.Errorf("no keys to save"), nil)
		return fmt.Errorf("no keys to save")
	}

	// Create directory if it doesn't exist
	keyDir := filepath.Dir(keyPath)
	if err := os.MkdirAll(keyDir, 0700); err != nil {
		sm.logger.Error("Failed to create key directory", err, map[string]interface{}{
			"directory": keyDir,
		})
		return fmt.Errorf("failed to create key directory: %w", err)
	}

	// Save private key
	privateKeyData := base64.StdEncoding.EncodeToString(sm.privateKey)
	privateKeyPath := keyPath + ".private"
	if err := os.WriteFile(privateKeyPath, []byte(privateKeyData), 0600); err != nil {
		sm.logger.Error("Failed to save private key", err, map[string]interface{}{
			"path": privateKeyPath,
		})
		return fmt.Errorf("failed to save private key: %w", err)
	}

	// Save public key
	publicKeyData := base64.StdEncoding.EncodeToString(sm.publicKey)
	publicKeyPath := keyPath + ".public"
	if err := os.WriteFile(publicKeyPath, []byte(publicKeyData), 0644); err != nil {
		sm.logger.Error("Failed to save public key", err, map[string]interface{}{
			"path": publicKeyPath,
		})
		return fmt.Errorf("failed to save public key: %w", err)
	}

	sm.logger.Info("Keys saved successfully", map[string]interface{}{
		"private_key_path": privateKeyPath,
		"public_key_path":  publicKeyPath,
	})

	return nil
}

// SignMessage signs a message with the private key
func (sm *SecurityManager) SignMessage(msg interface{}) (string, error) {
	if sm.privateKey == nil {
		sm.logger.Error("Private key not loaded", fmt.Errorf("private key not loaded"), nil)
		return "", fmt.Errorf("private key not loaded")
	}

	sm.logger.Debug("Signing message", map[string]interface{}{
		"message_type": fmt.Sprintf("%T", msg),
	})

	// Serialize the message to JSON for signing
	data, err := json.Marshal(msg)
	if err != nil {
		sm.logger.Error("Failed to serialize message", err, nil)
		return "", fmt.Errorf("failed to serialize message: %w", err)
	}

	// Sign the message
	signature := ed25519.Sign(sm.privateKey, data)

	// Return base64 encoded signature
	signatureStr := base64.StdEncoding.EncodeToString(signature)
	sm.logger.Debug("Message signed successfully", map[string]interface{}{
		"signature_length": len(signature),
		"data_length":      len(data),
	})

	return signatureStr, nil
}

// VerifyMessage verifies a message signature with the public key
func (sm *SecurityManager) VerifyMessage(msg interface{}, signature string) error {
	if sm.publicKey == nil {
		return fmt.Errorf("public key not loaded")
	}

	// Decode the signature
	sigBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return fmt.Errorf("failed to decode signature: %w", err)
	}

	// Serialize the message to JSON for verification
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Verify the signature
	if !ed25519.Verify(sm.publicKey, data, sigBytes) {
		return fmt.Errorf("signature verification failed")
	}

	return nil
}

// SignMeshMessage signs a mesh message and updates its signature field
func (sm *SecurityManager) SignMeshMessage(msg *internal.MeshMessage) error {
	signature, err := sm.SignMessage(msg)
	if err != nil {
		return err
	}

	msg.Signature = signature
	return nil
}

// VerifyMeshMessage verifies a mesh message signature
func (sm *SecurityManager) VerifyMeshMessage(msg *internal.MeshMessage) error {
	if msg.Signature == "" {
		return fmt.Errorf("message has no signature")
	}

	// Create a copy of the message without signature for verification
	msgCopy := *msg
	msgCopy.Signature = ""

	return sm.VerifyMessage(&msgCopy, msg.Signature)
}

// GetPublicKey returns the public key as base64 string
func (sm *SecurityManager) GetPublicKey() string {
	if sm.publicKey == nil {
		return ""
	}
	return base64.StdEncoding.EncodeToString(sm.publicKey)
}

// GetPublicKeyBytes returns the public key as bytes
func (sm *SecurityManager) GetPublicKeyBytes() []byte {
	if sm.publicKey == nil {
		return nil
	}
	return sm.publicKey
}

// HasKeys returns true if keys are loaded
func (sm *SecurityManager) HasKeys() bool {
	return sm.privateKey != nil && sm.publicKey != nil
}

// loadExistingKeys loads existing keys from the key path
func (sm *SecurityManager) loadExistingKeys() error {
	// Load private key
	privateKeyPath := sm.keyPath + ".private"
	privateKeyData, err := os.ReadFile(privateKeyPath)
	if err != nil {
		return err
	}

	privateKeyBytes, err := base64.StdEncoding.DecodeString(string(privateKeyData))
	if err != nil {
		return fmt.Errorf("failed to decode private key: %w", err)
	}

	// Load public key
	publicKeyPath := sm.keyPath + ".public"
	publicKeyData, err := os.ReadFile(publicKeyPath)
	if err != nil {
		return err
	}

	publicKeyBytes, err := base64.StdEncoding.DecodeString(string(publicKeyData))
	if err != nil {
		return fmt.Errorf("failed to decode public key: %w", err)
	}

	// Validate key lengths
	if len(privateKeyBytes) != ed25519.PrivateKeySize {
		return fmt.Errorf("invalid private key size: %d", len(privateKeyBytes))
	}
	if len(publicKeyBytes) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: %d", len(publicKeyBytes))
	}

	sm.privateKey = privateKeyBytes
	sm.publicKey = publicKeyBytes

	return nil
}

// generateAndSaveKeys generates new keys and saves them
func (sm *SecurityManager) generateAndSaveKeys() error {
	if err := sm.GenerateKeyPair(); err != nil {
		return err
	}

	return sm.SaveKeys(sm.keyPath)
}

// CreateSignedCommandMessage creates a signed command message
func (sm *SecurityManager) CreateSignedCommandMessage(
	command string,
	arguments []string,
	target []string,
	sender string,
	workDir string,
	timeout int,
) (*internal.CommandMessage, error) {
	// Create the message
	msg := &internal.CommandMessage{
		MeshMessage: internal.MeshMessage{
			ID:        sm.generateMessageID(),
			TTL:       5,
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

	// Sign the message
	if err := sm.SignMeshMessage(&msg.MeshMessage); err != nil {
		return nil, err
	}

	return msg, nil
}

// CreateSignedResultMessage creates a signed result message
func (sm *SecurityManager) CreateSignedResultMessage(
	commandID string,
	result internal.ExecutionResult,
	sender string,
) (*internal.ResultMessage, error) {
	// Create the message
	msg := &internal.ResultMessage{
		MeshMessage: internal.MeshMessage{
			ID:        sm.generateMessageID(),
			TTL:       3,
			Sender:    sender,
			Target:    []string{},
			Type:      internal.MessageTypeResult,
			Timestamp: time.Now().Unix(),
		},
		CommandID: commandID,
		Result:    result,
	}

	// Sign the message
	if err := sm.SignMeshMessage(&msg.MeshMessage); err != nil {
		return nil, err
	}

	return msg, nil
}

// generateMessageID generates a unique message ID
func (sm *SecurityManager) generateMessageID() string {
	// Use timestamp + random bytes for uniqueness
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	if _, err := rand.Read(randomBytes); err != nil {
		// Fallback to timestamp-only ID if entropy unavailable
		return fmt.Sprintf("%d", timestamp)
	}

	// Combine timestamp and random bytes
	id := fmt.Sprintf("%d-%x", timestamp, randomBytes)
	return id
}
