package targeting

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/monster0506/meshexec/internal"
	"github.com/monster0506/meshexec/internal/logging"
)

// Evaluator implements the TargetEvaluator interface
type Evaluator struct {
	logger *logging.Logger
}

// NewEvaluator creates a new target expression evaluator
func NewEvaluator() *Evaluator {
	return &Evaluator{
		logger: logging.NewLogger("info"),
	}
}

// NewEvaluatorWithLevel creates a new evaluator with a configurable log level.
// Useful for tests where logs should be silenced by passing level "none".
func NewEvaluatorWithLevel(level string) *Evaluator {
    return &Evaluator{
        logger: logging.NewLogger(level),
    }
}

// Evaluate evaluates a target expression against device information
func (e *Evaluator) Evaluate(expression string, device *internal.DeviceInfo) (bool, error) {
	if device == nil {
		e.logger.Error("Device info is nil", fmt.Errorf("device info cannot be nil"), nil)
		return false, fmt.Errorf("device info cannot be nil")
	}

	// Handle special case "all"
	if strings.TrimSpace(strings.ToLower(expression)) == "all" {
		e.logger.Debug("Evaluating 'all' target", map[string]interface{}{
			"expression": expression,
			"device":     device.Name,
		})
		return true, nil
	}

	e.logger.Debug("Evaluating target expression", map[string]interface{}{
		"expression":  expression,
		"device":      device.Name,
		"device_os":   device.OS,
		"device_arch": device.Arch,
	})

	// Parse the expression into an AST
	ast, err := e.Parse(expression)
	if err != nil {
		e.logger.Error("Failed to parse expression", err, map[string]interface{}{
			"expression": expression,
		})
		return false, fmt.Errorf("failed to parse expression: %w", err)
	}

	// Evaluate the AST
	result, err := e.evaluateAST(ast, device)
	if err != nil {
		e.logger.Error("Failed to evaluate AST", err, map[string]interface{}{
			"expression": expression,
		})
		return false, err
	}

	e.logger.Debug("Target evaluation result", map[string]interface{}{
		"expression": expression,
		"result":     result,
		"device":     device.Name,
	})

	return result, nil
}

// Parse parses a target expression into an abstract syntax tree
func (e *Evaluator) Parse(expression string) (*internal.TargetAST, error) {
	if strings.TrimSpace(expression) == "" {
		return nil, fmt.Errorf("expression cannot be empty")
	}

	// Tokenize the expression
	tokens, err := e.Tokenize(expression)
	if err != nil {
		return nil, fmt.Errorf("failed to tokenize expression: %w", err)
	}

	// Parse the tokens into an AST
	ast, err := e.parseTokens(tokens)
	if err != nil {
		return nil, fmt.Errorf("failed to parse tokens: %w", err)
	}

	return ast, nil
}

// Tokenize converts an expression string into tokens
func (e *Evaluator) Tokenize(expression string) ([]string, error) {
	var tokens []string
	var current strings.Builder
	inQuotes := false
	escapeNext := false

	for i, char := range expression {
		if escapeNext {
			current.WriteRune(char)
			escapeNext = false
			continue
		}

		if char == '\\' {
			escapeNext = true
			continue
		}

		if char == '"' {
			inQuotes = !inQuotes
			continue
		}

		if inQuotes {
			current.WriteRune(char)
			continue
		}

		// Handle operators
		if char == '&' && i+1 < len(expression) && expression[i+1] == '&' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.TrimSpace(current.String()))
				current.Reset()
			}
			tokens = append(tokens, "&&")
			i++ // Skip next character
			continue
		}

		if char == '|' && i+1 < len(expression) && expression[i+1] == '|' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.TrimSpace(current.String()))
				current.Reset()
			}
			tokens = append(tokens, "||")
			i++ // Skip next character
			continue
		}

		// Skip single & and | characters that are part of && and ||
		if char == '&' || char == '|' {
			continue
		}

		if char == '!' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.TrimSpace(current.String()))
				current.Reset()
			}
			tokens = append(tokens, "!")
			continue
		}

		if char == '(' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.TrimSpace(current.String()))
				current.Reset()
			}
			tokens = append(tokens, "(")
			continue
		}

		if char == ')' {
			if current.Len() > 0 {
				tokens = append(tokens, strings.TrimSpace(current.String()))
				current.Reset()
			}
			tokens = append(tokens, ")")
			continue
		}

		// Handle whitespace
		if unicode.IsSpace(char) {
			if current.Len() > 0 {
				tokens = append(tokens, strings.TrimSpace(current.String()))
				current.Reset()
			}
			continue
		}

		current.WriteRune(char)
	}

	if current.Len() > 0 {
		tokens = append(tokens, strings.TrimSpace(current.String()))
	}

	// Filter out empty tokens
	var filtered []string
	for _, token := range tokens {
		if token != "" {
			filtered = append(filtered, token)
		}
	}

	return filtered, nil
}

// parseTokens converts tokens into an abstract syntax tree
func (e *Evaluator) parseTokens(tokens []string) (*internal.TargetAST, error) {
	if len(tokens) == 0 {
		return nil, fmt.Errorf("no tokens to parse")
	}

	// Handle negation
	if tokens[0] == "!" {
		if len(tokens) < 2 {
			return nil, fmt.Errorf("incomplete negation expression")
		}
		operand, err := e.parseTokens(tokens[1:])
		if err != nil {
			return nil, err
		}
		return &internal.TargetAST{
			Type:     "unary",
			Operator: "!",
			Right:    operand,
		}, nil
	}

	// Handle single token (must be a condition)
	if len(tokens) == 1 {
		return e.parseCondition(tokens[0])
	}

	// Handle parentheses
	if tokens[0] == "(" {
		// Find matching closing parenthesis
		parenCount := 1
		closeIndex := -1
		for i := 1; i < len(tokens); i++ {
			if tokens[i] == "(" {
				parenCount++
			} else if tokens[i] == ")" {
				parenCount--
				if parenCount == 0 {
					closeIndex = i
					break
				}
			}
		}
		if closeIndex == -1 {
			return nil, fmt.Errorf("unmatched opening parenthesis")
		}

		// Parse the content inside parentheses
		content, err := e.parseTokens(tokens[1:closeIndex])
		if err != nil {
			return nil, err
		}

		// If there are more tokens after the closing parenthesis, handle them
		if closeIndex+1 < len(tokens) {
			remaining := tokens[closeIndex+1:]
			return e.parseBinaryExpression(content, remaining)
		}

		return content, nil
	}

	// Check for unmatched closing parenthesis at the top level
	parenCount := 0
	for _, token := range tokens {
		if token == "(" {
			parenCount++
		} else if token == ")" {
			parenCount--
			if parenCount < 0 {
				return nil, fmt.Errorf("unmatched closing parenthesis")
			}
		}
	}
	if parenCount > 0 {
		return nil, fmt.Errorf("unmatched opening parenthesis")
	}

	// Handle binary expressions (&&, ||)
	return e.parseBinaryExpression(nil, tokens)
}

// parseBinaryExpression handles binary operators (&&, ||)
func (e *Evaluator) parseBinaryExpression(left *internal.TargetAST, tokens []string) (*internal.TargetAST, error) {
	if len(tokens) == 0 {
		return left, nil
	}

	// Find the lowest precedence operator (|| has lower precedence than &&)
	lowestPrec := -1
	lowestIndex := -1

	parenCount := 0
	for i, token := range tokens {
		if token == "(" {
			parenCount++
		} else if token == ")" {
			parenCount--
		} else if parenCount == 0 { // Only consider operators outside parentheses
			if token == "||" {
				lowestPrec = 0
				lowestIndex = i
			} else if token == "&&" && lowestPrec < 0 {
				lowestPrec = 1
				lowestIndex = i
			}
		}
	}

	if lowestIndex == -1 {
		// No binary operator found, must be a single condition
		if left != nil {
			return nil, fmt.Errorf("unexpected tokens after expression")
		}
		if len(tokens) != 1 {
			return nil, fmt.Errorf("invalid expression: multiple tokens without operator")
		}
		return e.parseCondition(tokens[0])
	}

	// Parse left operand
	var leftOperand *internal.TargetAST
	var err error
	if left != nil {
		leftOperand = left
	} else {
		leftOperand, err = e.parseTokens(tokens[:lowestIndex])
		if err != nil {
			return nil, err
		}
	}

	// Parse right operand
	rightTokens := tokens[lowestIndex+1:]
	rightOperand, err := e.parseTokens(rightTokens)
	if err != nil {
		return nil, err
	}

	return &internal.TargetAST{
		Type:     "binary",
		Operator: tokens[lowestIndex],
		Left:     leftOperand,
		Right:    rightOperand,
	}, nil
}

// parseCondition parses a single condition (e.g., "os=linux", "role=worker")
func (e *Evaluator) parseCondition(condition string) (*internal.TargetAST, error) {
	// Validate condition format: attribute=value
	parts := strings.SplitN(condition, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid condition format: %s (expected attribute=value)", condition)
	}

	attribute := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	if attribute == "" {
		return nil, fmt.Errorf("attribute cannot be empty in condition: %s", condition)
	}

	if value == "" {
		return nil, fmt.Errorf("invalid condition format: %s (expected attribute=value)", condition)
	}

	// Remove quotes from value if present
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
		value = value[1 : len(value)-1]
	}

	return &internal.TargetAST{
		Type:  "condition",
		Value: condition,
	}, nil
}

// evaluateAST evaluates an abstract syntax tree against device information
func (e *Evaluator) evaluateAST(ast *internal.TargetAST, device *internal.DeviceInfo) (bool, error) {
	if ast == nil {
		return false, fmt.Errorf("AST cannot be nil")
	}

	switch ast.Type {
	case "condition":
		return e.evaluateCondition(ast.Value, device)
	case "unary":
		if ast.Operator == "!" {
			result, err := e.evaluateAST(ast.Right, device)
			if err != nil {
				return false, err
			}
			return !result, nil
		}
		return false, fmt.Errorf("unknown unary operator: %s", ast.Operator)
	case "binary":
		leftResult, err := e.evaluateAST(ast.Left, device)
		if err != nil {
			return false, err
		}

		rightResult, err := e.evaluateAST(ast.Right, device)
		if err != nil {
			return false, err
		}

		switch ast.Operator {
		case "&&":
			return leftResult && rightResult, nil
		case "||":
			return leftResult || rightResult, nil
		default:
			return false, fmt.Errorf("unknown binary operator: %s", ast.Operator)
		}
	default:
		return false, fmt.Errorf("unknown AST node type: %s", ast.Type)
	}
}

// evaluateCondition evaluates a single condition against device information
func (e *Evaluator) evaluateCondition(condition string, device *internal.DeviceInfo) (bool, error) {
	parts := strings.SplitN(condition, "=", 2)
	if len(parts) != 2 {
		return false, fmt.Errorf("invalid condition format: %s", condition)
	}

	attribute := strings.TrimSpace(parts[0])
	value := strings.TrimSpace(parts[1])

	// Remove quotes from value if present
	if len(value) >= 2 && (value[0] == '"' && value[len(value)-1] == '"') {
		value = value[1 : len(value)-1]
	}

	// Get the device attribute value
	var deviceValue string
	switch strings.ToLower(attribute) {
	case "name":
		deviceValue = device.Name
	case "role":
		deviceValue = device.Role
	case "os":
		deviceValue = device.OS
	case "arch":
		deviceValue = device.Arch
	default:
		// Check if it's a tag
		if device.Tags != nil {
			if tagValue, exists := device.Tags[attribute]; exists {
				deviceValue = tagValue
			} else {
				return false, nil // Tag doesn't exist, condition fails
			}
		} else {
			return false, nil // No tags, condition fails
		}
	}

	// Compare values (case-insensitive for better usability)
	return strings.EqualFold(deviceValue, value), nil
}
