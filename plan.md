# **MeshExec CLI - Tech Spec**

## **Overview**

MeshExec CLI allows multiple Bluetooth-enabled devices to form a self-healing mesh network to broadcast, relay, and execute shell commands securely and efficiently without Wi-Fi or central infrastructure.

---

## **1. Goals**

* Decentralized shell command execution
* Bluetooth LE mesh network (no Wi-Fi)
* Role-based device discovery and targeting
* Secure message passing
* Low-latency, resilient command propagation
* TUI and CLI interfaces
* Extensible for file transfers and synchronization

---

## **2. System Architecture**

```
             +-------------------+     +-------------------+
             |    Device A       |<--->|     Device B      |
             | meshexec agent    |     | meshexec agent    |
             +-------------------+     +-------------------+
                      ^                         ^
                      |                         |
                      v                         v
             +-------------------+     +-------------------+
             |    Device C       |<--->|     Device D      |
             | meshexec agent    |     | meshexec agent    |
             +-------------------+     +-------------------+
```

---

## **3. Components**

### **A. CLI Frontend (`meshexec`)**

* Written in Go using [Cobra](https://github.com/spf13/cobra)
* Subcommands:

  * `run` - Send a command to the mesh
  * `join` - Start advertising and listening
  * `list` - Show connected peers
  * `status` - Show execution status
  * `tui` - Launch terminal UI dashboard

---

### **B. Mesh Network Layer**

* Bluetooth LE GATT-based peer-to-peer link
* Mesh messages are relayed (flooding with TTL)
* Custom protocol packet format

---

### **C. Agent Daemon**

* Listens for commands over BLE
* Executes commands with system shell
* Sends result/output back as BLE packets
* Optional filtering/tagging of device roles

---

### **D. Packet Format**

```json
{
  "id": "uuid",
  "ttl": 3,
  "sender": "device-xyz",
  "target": ["linux", "role=worker"],
  "type": "cmd",
  "command": "uptime",
  "signature": "base64(sig)",
  "timestamp": 1723043812
}
```

---

### **E. Execution Result**

```json
{
  "id": "uuid",
  "type": "result",
  "status": "success",
  "stdout": "10:41 up 5 min,  2 users,  load average: 0.12, 0.08, 0.01",
  "stderr": "",
  "code": 0,
  "device": "device-xyz"
}
```

---

## **4. Core Features**

| Feature                  | Description                      |
| ------------------------ | -------------------------------- |
| Command Dispatch         | Send shell commands across mesh  |
| Peer Discovery           | Auto-discovery of nearby devices |
| Command Filtering        | By tag, architecture, or OS      |
| TTL                      | Prevent flooding loops           |
| Signature Verification   | Command authenticity via ed25519 |
| Encrypted Payloads (opt) | AES-GCM + pre-shared key         |
| Result Aggregation       | Collect outputs and show in TUI  |
| File Transfer (planned)  | Base64 chunks over mesh          |

---

## **5. Targeting Syntax**

Supports expressions like:

```
--target="os=linux && role=worker"
--target="!arch=arm"
--target="all"
```

---

## **6. Example CLI Commands**

```sh
# Basic command
meshexec run --target="all" "uptime"

# Scoped execution
meshexec run --target="arch=arm" "cat /proc/cpuinfo"

# Role-based
meshexec run --target="role=logger" "logrotate"

# Check who’s alive
meshexec list

# Get the last 10 logs from all devices
meshexec run "tail -n 10 /var/log/syslog"

# Reboot all field units
meshexec run --target="zone=alpha" "sudo reboot"

# Chain multiple commands
meshexec run 'cd /opt/service && git pull && systemctl restart api'

# Dry-run mode
meshexec run --dry-run "rm -rf /"

# Start the agent listener
meshexec join

# Launch dashboard
meshexec tui
```

---

## **7. Go Code Skeleton**

### **Command Dispatch (simplified)**

```go
func SendCommand(command string, target string) {
	msg := MeshMessage{
		ID:        uuid.New().String(),
		TTL:       5,
		Sender:    hostname(),
		Type:      "cmd",
		Command:   command,
		Target:    ParseTarget(target),
		Timestamp: time.Now().Unix(),
	}
	msg.Sign(privateKey)

	SendToMesh(msg)
}
```

### **BLE GATT Mesh Node**

```go
func StartMeshNode() {
	go Advertise()
	for {
		packet := ReceiveBLE()
		if packet.IsExpired() || packet.IsDuplicate() {
			continue
		}
		if ShouldRelay(packet) {
			Relay(packet)
		}
		if packet.Type == "cmd" && TargetMatch(packet.Target) {
			result := ExecuteShell(packet.Command)
			SendResult(packet.ID, result)
		}
	}
}
```

### **Command Execution**

```go
func ExecuteShell(cmd string) ExecutionResult {
	c := exec.Command("/bin/sh", "-c", cmd)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	err := c.Run()

	return ExecutionResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Code:   c.ProcessState.ExitCode(),
		Status: "success",
	}
}
```

---

## **8. Security Model**

* **Command Signatures**: ed25519 keys for sender verification
* **Encrypted Commands**: Optional AES-GCM over mesh messages
* **Allow/Deny Lists**: Devices can reject unknown or unsafe commands
* **Safe Mode**: Filter out dangerous operations (e.g., no `rm -rf`)

---

## **9. Testing Strategy**

* **Simulated Mesh** in-memory for unit tests
* **BLE Mocking Layer** for integration
* **Dockerized BLE MeshExec** for multi-node testing on one machine
* **Log Replay System** to simulate real environments

---

## **10. Stretch Goals**

* **Interactive Shell Sessions** over mesh
* **Remote File Sync** (diff-based)
* **Event Subscriptions** (e.g. log when CPU > 80%)
* **Physical UI**: LED blink feedback per device on success/fail
* **WebSocket Proxy** to bridge mesh to internet

---

Want to dive into the actual Bluetooth mesh protocol implementation in Go next (e.g., advertising, scanning, connection queueing, relaying packets, GATT services)? Or prefer to work on the CLI structure first?
