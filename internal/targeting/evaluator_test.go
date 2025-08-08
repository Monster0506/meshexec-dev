package targeting

import (
	"testing"

	"github.com/monster0506/meshexec/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewEvaluator(t *testing.T) {
	evaluator := NewEvaluator()
	assert.NotNil(t, evaluator)
}

func TestEvaluator_Evaluate_All(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
	}

	// Test "all" target
	result, err := evaluator.Evaluate("all", device)
	require.NoError(t, err)
	assert.True(t, result)

	// Test "ALL" (case insensitive)
	result, err = evaluator.Evaluate("ALL", device)
	require.NoError(t, err)
	assert.True(t, result)

	// Test " All " (with whitespace)
	result, err = evaluator.Evaluate(" All ", device)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_Evaluate_SingleConditions(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: map[string]string{
			"environment": "production",
			"location":    "datacenter",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"os match", "os=linux", true},
		{"os mismatch", "os=windows", false},
		{"role match", "role=worker", true},
		{"role mismatch", "role=manager", false},
		{"arch match", "arch=amd64", true},
		{"arch mismatch", "arch=arm", false},
		{"name match", "name=test-device", true},
		{"name mismatch", "name=other-device", false},
		{"tag match", "environment=production", true},
		{"tag mismatch", "environment=development", false},
		{"nonexistent tag", "nonexistent=value", false},
		{"case insensitive os", "OS=LINUX", true},
		{"case insensitive role", "ROLE=WORKER", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_AndExpressions(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: map[string]string{
			"environment": "production",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"both true", "os=linux && role=worker", true},
		{"first false", "os=windows && role=worker", false},
		{"second false", "os=linux && role=manager", false},
		{"both false", "os=windows && role=manager", false},
		{"with tag", "os=linux && environment=production", true},
		{"tag mismatch", "os=linux && environment=development", false},
		{"multiple conditions", "os=linux && role=worker && arch=amd64", true},
		{"one mismatch in multiple", "os=linux && role=worker && arch=arm", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_OrExpressions(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: map[string]string{
			"environment": "production",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"both true", "os=linux || role=worker", true},
		{"first true", "os=linux || role=manager", true},
		{"second true", "os=windows || role=worker", true},
		{"both false", "os=windows || role=manager", false},
		{"with tag", "os=windows || environment=production", true},
		{"tag mismatch", "os=windows || environment=development", false},
		{"multiple conditions", "os=windows || role=manager || arch=amd64", true},
		{"all false", "os=windows || role=manager || arch=arm", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_NotExpressions(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: map[string]string{
			"environment": "production",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"not true condition", "!os=windows", true},
		{"not false condition", "!os=linux", false},
		{"not role", "!role=manager", true},
		{"not arch", "!arch=arm", true},
		{"not tag", "!environment=development", true},
		{"not existing tag", "!nonexistent=value", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_ComplexExpressions(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: map[string]string{
			"environment": "production",
			"location":    "datacenter",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"and or combination", "os=linux && (role=worker || role=manager)", true},
		{"or and combination", "os=windows || (role=worker && arch=amd64)", true},
		{"not and combination", "!os=windows && role=worker", true},
		{"not or combination", "!os=windows || role=manager", true},
		{"complex nested", "(os=linux && role=worker) || (os=windows && role=manager)", true},
		{"complex with tags", "(os=linux && environment=production) || (role=manager && location=datacenter)", true},
		{"multiple parentheses", "((os=linux && role=worker) || (arch=amd64 && environment=production))", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_QuotedValues(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: map[string]string{
			"environment": "production",
		},
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"quoted value", `os="linux"`, true},
		{"quoted value mismatch", `os="windows"`, false},
		{"quoted role", `role="worker"`, true},
		{"quoted tag", `environment="production"`, true},
		{"quoted with spaces", `os="linux" && role="worker"`, true},
		{"mixed quoted unquoted", `os="linux" && role=worker`, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_WhitespaceHandling(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
	}

	tests := []struct {
		name       string
		expression string
		expected   bool
	}{
		{"extra spaces", "  os=linux  &&  role=worker  ", true},
		{"tabs and spaces", "os=linux\t&&\trole=worker", true},
		{"newlines", "os=linux\n&&\nrole=worker", true},
		{"mixed whitespace", "  os=linux \t && \n role=worker  ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Evaluate(tt.expression, device)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Evaluate_ErrorCases(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
	}

	tests := []struct {
		name        string
		expression  string
		device      *internal.DeviceInfo
		expectError bool
		errorMsg    string
	}{
		{"nil device", "os=linux", nil, true, "device info cannot be nil"},
		{"empty expression", "", device, true, "expression cannot be empty"},
		{"whitespace only", "   ", device, true, "expression cannot be empty"},
		{"invalid condition format", "os", device, true, "invalid condition format"},
		{"missing value", "os=", device, true, "invalid condition format"},
		{"missing attribute", "=linux", device, true, "attribute cannot be empty"},
		{"incomplete negation", "!", device, true, "incomplete negation expression"},
		{"unmatched opening parenthesis", "(os=linux", device, true, "unmatched opening parenthesis"},
		{"unmatched closing parenthesis", "os=linux)", device, true, "unmatched closing parenthesis"},
		{"multiple tokens without operator", "os=linux role=worker", device, true, "invalid expression: multiple tokens without operator"},
	}

			for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evaluator.Evaluate(tt.expression, tt.device)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestEvaluator_Parse(t *testing.T) {
	evaluator := NewEvaluator()

	tests := []struct {
		name       string
		expression string
		expected   *internal.TargetAST
	}{
		{
			name:       "simple condition",
			expression: "os=linux",
			expected: &internal.TargetAST{
				Type:  "condition",
				Value: "os=linux",
			},
		},
		{
			name:       "and expression",
			expression: "os=linux && role=worker",
			expected: &internal.TargetAST{
				Type:     "binary",
				Operator: "&&",
				Left: &internal.TargetAST{
					Type:  "condition",
					Value: "os=linux",
				},
				Right: &internal.TargetAST{
					Type:  "condition",
					Value: "role=worker",
				},
			},
		},
		{
			name:       "or expression",
			expression: "os=linux || role=worker",
			expected: &internal.TargetAST{
				Type:     "binary",
				Operator: "||",
				Left: &internal.TargetAST{
					Type:  "condition",
					Value: "os=linux",
				},
				Right: &internal.TargetAST{
					Type:  "condition",
					Value: "role=worker",
				},
			},
		},
		{
			name:       "not expression",
			expression: "!os=windows",
			expected: &internal.TargetAST{
				Type:     "unary",
				Operator: "!",
				Right: &internal.TargetAST{
					Type:  "condition",
					Value: "os=windows",
				},
			},
		},
		{
			name:       "parentheses",
			expression: "(os=linux && role=worker)",
			expected: &internal.TargetAST{
				Type:     "binary",
				Operator: "&&",
				Left: &internal.TargetAST{
					Type:  "condition",
					Value: "os=linux",
				},
				Right: &internal.TargetAST{
					Type:  "condition",
					Value: "role=worker",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := evaluator.Parse(tt.expression)
			require.NoError(t, err)
			assertASTEqual(t, tt.expected, result)
		})
	}
}

func TestEvaluator_Parse_ErrorCases(t *testing.T) {
	evaluator := NewEvaluator()

	tests := []struct {
		name        string
		expression  string
		expectError bool
		errorMsg    string
	}{
		{"empty expression", "", true, "expression cannot be empty"},
		{"whitespace only", "   ", true, "expression cannot be empty"},
		{"invalid condition", "os", true, "invalid condition format"},
		{"incomplete negation", "!", true, "incomplete negation expression"},
		{"unmatched parenthesis", "(os=linux", true, "unmatched opening parenthesis"},
		{"no tokens", "", true, "expression cannot be empty"},
	}

			for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := evaluator.Parse(tt.expression)
			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Helper function to compare ASTs
func assertASTEqual(t *testing.T, expected, actual *internal.TargetAST) {
	if expected == nil && actual == nil {
		return
	}
	if expected == nil || actual == nil {
		t.Errorf("AST mismatch: expected %v, got %v", expected, actual)
		return
	}

	assert.Equal(t, expected.Type, actual.Type)
	assert.Equal(t, expected.Operator, actual.Operator)
	assert.Equal(t, expected.Value, actual.Value)

	if expected.Left != nil || actual.Left != nil {
		assertASTEqual(t, expected.Left, actual.Left)
	}
	if expected.Right != nil || actual.Right != nil {
		assertASTEqual(t, expected.Right, actual.Right)
	}
}

func TestEvaluator_DeviceWithNilTags(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
		Tags: nil, // Explicitly nil tags
	}

	// Test that tag conditions fail gracefully when tags are nil
	result, err := evaluator.Evaluate("environment=production", device)
	require.NoError(t, err)
	assert.False(t, result)

	// Test that non-tag conditions still work
	result, err = evaluator.Evaluate("os=linux", device)
	require.NoError(t, err)
	assert.True(t, result)
}

func TestEvaluator_OperatorPrecedence(t *testing.T) {
	evaluator := NewEvaluator()
	device := &internal.DeviceInfo{
		Name: "test-device",
		Role: "worker",
		OS:   "linux",
		Arch: "amd64",
	}

	// Test that && has higher precedence than ||
	// "os=linux || role=worker && arch=arm" should be equivalent to "os=linux || (role=worker && arch=arm)"
	// Since role=worker is true but arch=arm is false, the && evaluates to false
	// So the whole expression should be true (os=linux is true)
	result, err := evaluator.Evaluate("os=linux || role=worker && arch=arm", device)
	require.NoError(t, err)
	assert.True(t, result)

	// Test with parentheses to override precedence
	// "(os=linux || role=worker) && arch=arm" should be false
	result, err = evaluator.Evaluate("(os=linux || role=worker) && arch=arm", device)
	require.NoError(t, err)
	assert.False(t, result)
}



	

	

 