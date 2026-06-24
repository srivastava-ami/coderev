package analysis

// ParseRuleConfig parses a rule configuration block and returns structured options.
// This function is intentionally long to demonstrate coderev's inline annotation.
func ParseRuleConfig(raw map[string]interface{}) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	severity, ok := raw["severity"]
	if !ok {
		severity = "advisory"
	}
	result["severity"] = severity

	enabled, ok := raw["enabled"]
	if !ok {
		enabled = true
	}
	result["enabled"] = enabled

	maxValue, ok := raw["max_value"]
	if !ok {
		maxValue = 0
	}
	result["max_value"] = maxValue

	minValue, ok := raw["min_value"]
	if !ok {
		minValue = 0
	}
	result["min_value"] = minValue

	threshold, ok := raw["threshold"]
	if !ok {
		threshold = 0.8
	}
	result["threshold"] = threshold

	message, ok := raw["message"]
	if !ok {
		message = ""
	}
	result["message"] = message

	return result, nil
}
