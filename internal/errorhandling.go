package internal

import (
	"errors"
	"fmt"
	"strings"
)

// NewNetworkError constructs a MeshExecError of type network.
func NewNetworkError(code string, message string, details map[string]interface{}) *MeshExecError {
	return &MeshExecError{Type: ErrorTypeNetwork, Code: code, Message: message, Details: cloneDetails(details)}
}

// NewExecutionError constructs a MeshExecError of type execution.
func NewExecutionError(code string, message string, details map[string]interface{}) *MeshExecError {
	return &MeshExecError{Type: ErrorTypeExecution, Code: code, Message: message, Details: cloneDetails(details)}
}

// NewSecurityError constructs a MeshExecError of type security.
func NewSecurityError(code string, message string, details map[string]interface{}) *MeshExecError {
	return &MeshExecError{Type: ErrorTypeSecurity, Code: code, Message: message, Details: cloneDetails(details)}
}

// NewConfigError constructs a MeshExecError of type config.
func NewConfigError(code string, message string, details map[string]interface{}) *MeshExecError {
	return &MeshExecError{Type: ErrorTypeConfig, Code: code, Message: message, Details: cloneDetails(details)}
}

// NewTargetingError constructs a MeshExecError of type targeting.
func NewTargetingError(code string, message string, details map[string]interface{}) *MeshExecError {
	return &MeshExecError{Type: ErrorTypeTargeting, Code: code, Message: message, Details: cloneDetails(details)}
}

// WrapAs converts any error to a MeshExecError with the provided metadata.
// If err is already a MeshExecError, returns it unchanged unless a non-empty
// code or message is provided, in which case they override the originals.
func WrapAs(err error, typ ErrorType, code string, message string, details map[string]interface{}) *MeshExecError {
	if err == nil {
		return nil
	}
	if me, ok := err.(*MeshExecError); ok {
		out := &MeshExecError{Type: me.Type, Code: me.Code, Message: me.Message, Details: cloneDetails(me.Details)}
		if typ != "" {
			out.Type = typ
		}
		if code != "" {
			out.Code = code
		}
		if message != "" {
			out.Message = message
		}
		mergeDetails(out, details)
		return out
	}
	if message == "" {
		message = err.Error()
	}
	return &MeshExecError{Type: typ, Code: code, Message: message, Details: cloneDetails(details)}
}

// FromError normalizes any error to a MeshExecError.
// Non-MeshExec errors default to execution type with code "generic_error".
func FromError(err error) *MeshExecError {
	if err == nil {
		return nil
	}
	if me, ok := err.(*MeshExecError); ok {
		return me
	}
	return &MeshExecError{Type: ErrorTypeExecution, Code: "generic_error", Message: err.Error(), Details: nil}
}

// FormatUserError produces a concise message for CLI display.
// Format: "[<type>:<code>] <message>". Details may be appended selectively.
func FormatUserError(err error) string {
	if err == nil {
		return ""
	}
	me := FromError(err)
	prefix := fmt.Sprintf("[%s:%s]", strings.ToLower(string(me.Type)), me.Code)
	var hint string
	switch me.Type {
	case ErrorTypeNetwork:
		switch me.Code {
		case "adapter_unavailable":
			hint = " Ensure Bluetooth is enabled and available."
		case "permission_denied":
			hint = " Check Bluetooth permissions."
		case "scan_failed":
			hint = " Try again or restart the adapter."
		}
	case ErrorTypeConfig:
		if me.Code == "invalid_config" {
			hint = " Fix the configuration file and retry."
		}
	case ErrorTypeSecurity:
		if me.Code == "signature_invalid" {
			hint = " Verify keys and signatures."
		}
	case ErrorTypeExecution:
		if me.Code == "dangerous_command" {
			hint = " Adjust safety settings or modify the command."
		}
	}
	return fmt.Sprintf("%s %s%s", prefix, me.Message, hint)
}

// MapNetworkError maps raw errors to structured MeshExecError codes.
// The operation indicates the BLE/network action (e.g., "scan", "advertise", "connect").
func MapNetworkError(operation string, err error) *MeshExecError {
	if err == nil {
		return nil
	}
	e := strings.ToLower(err.Error())
	if strings.Contains(e, "permission") {
		return NewNetworkError("permission_denied", fmt.Sprintf("%s permission denied", operation), map[string]interface{}{"operation": operation})
	}
	if strings.Contains(e, "not supported") || strings.Contains(e, "unsupported") {
		return NewNetworkError("unsupported", fmt.Sprintf("%s unsupported on this platform", operation), map[string]interface{}{"operation": operation})
	}
	if strings.Contains(e, "not available") || strings.Contains(e, "no adapter") || strings.Contains(e, "adapter") {
		return NewNetworkError("adapter_unavailable", fmt.Sprintf("%s failed: adapter unavailable", operation), map[string]interface{}{"operation": operation})
	}
	switch operation {
	case "scan":
		return NewNetworkError("scan_failed", "scan failed", map[string]interface{}{"error": err.Error()})
	case "advertise":
		return NewNetworkError("advertise_failed", "advertise failed", map[string]interface{}{"error": err.Error()})
	case "connect":
		return NewNetworkError("connect_failed", "connect failed", map[string]interface{}{"error": err.Error()})
	default:
		return NewNetworkError("network_error", err.Error(), nil)
	}
}

func cloneDetails(in map[string]interface{}) map[string]interface{} {
	if in == nil {
		return nil
	}
	out := make(map[string]interface{}, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func mergeDetails(me *MeshExecError, more map[string]interface{}) {
	if more == nil {
		return
	}
	if me.Details == nil {
		me.Details = make(map[string]interface{}, len(more))
	}
	for k, v := range more {
		me.Details[k] = v
	}
}

// IsMeshExecError reports whether err is a MeshExecError.
func IsMeshExecError(err error) bool {
	var me *MeshExecError
	return errors.As(err, &me)
}
