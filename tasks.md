# Implementation Plan

- [x] 1. Set up project structure and core interfaces
  - Create directory structure for cmd, internal packages (mesh, agent, config, targeting)
  - Define core interfaces for MeshNode, Agent, CommandExecutor, and ConfigManager
  - Set up go.mod with required dependencies (cobra, viper, toml, uuid, crypto, bubbletea, zerolog)
  - _Requirements: 1.1, 2.1, 8.1_

- [x] 2. Implement configuration management system
  - [x] 2.1 Create configuration data structures and parsing
    - Implement Config, DeviceConfig, SecurityConfig, NetworkConfig, and SafetyConfig structs
    - Write TOML file parsing using viper and BurntSushi/toml
    - Create unit tests for configuration parsing with sample config files
    - _Requirements: 8.1, 8.2, 8.3, 8.4_

  - [x] 2.2 Implement configuration manager with file watching
    - Write ConfigManager implementation with Load, Save, and Watch methods
    - Add cross-platform configuration file discovery (Windows: %APPDATA%, Unix: ~/.config, /etc, current dir)
    - Handle Windows-specific path separators and file permissions
    - Create unit tests for configuration loading and file watching on both platforms
    - _Requirements: 8.1, 8.2_

- [x] 3. Implement core data models and message handling
  - [x] 3.1 Create mesh message data structures
    - Implement MeshMessage, CommandMessage, ResultMessage, and ExecutionResult structs
    - Add JSON serialization/deserialization with proper field tags
    - Write unit tests for message serialization and validation
    - _Requirements: 1.1, 1.3, 5.1, 5.2_

  - [x] 3.2 Implement message signing and verification
    - Create SecurityManager interface and implementation using ed25519
    - Add message signing with private key and signature verification
    - Implement key generation, loading, and saving functionality
    - Write unit tests for cryptographic operations
    - _Requirements: 3.1, 3.2, 3.3_

- [x] 4. Implement target expression evaluation engine
  - [x] 4.1 Create target expression parser and evaluator
    - Implement TargetEvaluator interface with expression parsing logic
    - Support boolean expressions (&&, ||, !) and device attribute matching
    - Handle expressions like "os=linux && role=worker" and "!arch=arm"
    - Write unit tests for various target expression scenarios
    - _Requirements: 4.1, 4.2, 4.3, 4.4_

  - [x] 4.2 Implement device information matching
    - Create DeviceInfo struct with name, role, OS, arch, and tags
    - Implement device attribute evaluation against target expressions
    - Add support for "all" target and complex boolean logic
    - Write unit tests for device matching scenarios
    - _Requirements: 4.1, 4.2, 4.3_

- [x] 5. Implement command execution system
  - [x] 5.1 Create command executor with cross-platform shell integration
    - Implement CommandExecutor interface using os/exec package with Windows/Unix shell detection
    - Execute commands with cmd.exe on Windows and /bin/sh on Unix systems
    - Add platform-specific command handling and path resolution
    - Add command timeout handling and context cancellation
    - Write unit tests for command execution on both Windows and Unix platforms
    - _Requirements: 1.2, 1.3, 1.4_

  - [x] 5.2 Implement command safety and filtering
    - Add command validation for dangerous operations (rm -rf on Unix, del /s on Windows, etc.)
    - Implement platform-specific dangerous command detection (PowerShell, cmd.exe, bash)
    - Implement allow/deny list checking from configuration
    - Create safe mode filtering and dry-run command preview for both platforms
    - Write unit tests for command safety validation on Windows and Unix
    - _Requirements: 7.2, 7.3, 7.4, 7.1_

- [x] 6. Implement Bluetooth LE transport layer
  - [x] 6.1 Create BLE transport interface and basic operations
    - Implement BLETransport interface with Advertise, Scan, and Connect methods
    - Use native Go Bluetooth libraries (e.g., tinygo.org/x/bluetooth or go-ble/ble) for cross-platform support
    - Set up GATT service creation and characteristic handling using Go bindings
    - Add device discovery and connection management with unified Go interface
    - Write unit tests with mock BLE transport using Go interfaces
    - _Requirements: 2.1, 2.2, 2.3_

  - [x] 6.2 Implement BLE message transmission
    - Add message sending and receiving over GATT characteristics using Go BLE libraries
    - Implement message fragmentation for large payloads with cross-platform MTU handling
    - Handle BLE connection errors and reconnection logic through Go BLE abstractions
    - Add comprehensive error handling for adapter states using Go library error types
    - Write integration tests for BLE message transmission using Go BLE test utilities
    - _Requirements: 2.1, 2.3, 2.4_

- [ ] 7. Implement mesh networking layer
  - [x] 7.1 Create mesh node with peer management
    - Implement MeshNode interface with Start, Stop, SendMessage methods
    - Add peer discovery, connection tracking, and topology management
    - Implement message subscription system for different message types
    - Write unit tests for mesh node operations with mock transport
    - _Requirements: 2.1, 2.2, 2.3, 9.1, 9.2_

  - [ ] 7.2 Implement message routing and TTL handling
    - Add message routing logic with TTL decrementing and duplicate detection
    - Implement message relay functionality for multi-hop communication
    - Create routing table management and path discovery
    - Write unit tests for message routing scenarios
    - _Requirements: 5.1, 5.2, 5.3, 5.4, 9.1, 9.3_

  - [ ] 7.3 Implement network topology management
    - Add automatic route discovery and network healing
    - Implement peer status tracking and connection monitoring
    - Create network topology updates and peer information sharing
    - Write integration tests for network topology changes
    - _Requirements: 2.4, 9.3, 9.4_

- [x] 8. Implement agent daemon core functionality
  - [x] 8.1 Create agent with command processing
    - Implement Agent interface with Start, Stop, ProcessCommand methods
    - Add command message handling and execution coordination
    - Integrate with SecurityManager for message validation
    - Write unit tests for agent command processing
    - _Requirements: 1.1, 1.2, 3.2, 3.3_

  - [x] 8.2 Integrate agent with mesh networking and execution
    - Connect agent to mesh node for message sending and receiving
    - Add command execution using CommandExecutor and result handling
    - Implement target expression evaluation for incoming commands
    - Write integration tests for end-to-end command execution
    - _Requirements: 1.1, 1.2, 1.3, 4.3, 4.4_

- [ ] 9. Implement CLI frontend commands
  - [x] 9.1 Create basic CLI structure with Cobra
    - Set up main CLI application with cobra framework
    - Implement basic command structure (run, join, list, status, tui)
    - Add global flags and configuration loading
    - Write unit tests for CLI command parsing
    - _Requirements: 6.1, 6.2_

  - [ ] 9.2 Implement run command with targeting
    - Add run command implementation with target expression parsing
    - Integrate with agent for command execution and result collection
    - Add dry-run mode and command validation
    - Write unit tests for run command functionality
    - _Requirements: 1.1, 4.1, 4.2, 7.1_

  - [x] 9.3 Implement network management commands
    - Add join command to start mesh participation
    - Implement list command to show connected peers
    - Add status command for execution status display
    - Write unit tests for network management commands
    - _Requirements: 2.1, 6.1, 6.2, 6.3_

- [ ] 10. Implement terminal UI dashboard
  - [x] 10.1 Create TUI framework with bubbletea
    - Set up bubbletea-based terminal UI with multiple views
    - Implement real-time peer list and network status display
    - Add command execution results aggregation and display
    - Write unit tests for TUI components
    - _Requirements: 6.4_

  - [ ] 10.2 Integrate TUI with live data updates
    - Connect TUI to agent for real-time network and execution updates
    - Add interactive command execution from TUI interface
    - Implement result filtering and search functionality
    - Write integration tests for TUI data updates
    - _Requirements: 6.4_

- [x] 11. Add comprehensive error handling and logging
  - [x] 11.1 Implement structured error handling
    - Create MeshExecError types for different error categories
    - Add error handling strategies for network, execution, and security errors
    - Implement error propagation and user-friendly error messages
    - Write unit tests for error handling scenarios
    - _Requirements: 1.4, 3.3, 7.4, 9.4_

  - [x] 11.2 Implement structured logging system
    - Set up zerolog-based logging with configurable levels
    - Add logging throughout all components with appropriate levels
    - Implement log rotation and configuration options
    - Write tests for logging functionality
    - _Requirements: 3.3, 7.4_

- [ ] 12. Create comprehensive test suite
  - [ ] 12.1 Implement integration tests for mesh networking
    - Create test infrastructure with mock BLE transport and multiple nodes
    - Write tests for mesh formation, command propagation, and network healing
    - Add tests for security validation and configuration loading
    - Set up automated test execution with proper cleanup
    - _Requirements: All requirements validation_

  - [ ] 12.2 Create end-to-end testing framework
    - Implement Docker-based testing with multiple simulated devices
    - Add performance tests for message latency and throughput
    - Create security tests for signature verification and command safety
    - Write documentation for running and extending tests
    - _Requirements: All requirements validation_