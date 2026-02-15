package common

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v3"
)

// ParseConfig 解析配置内容到目标对象
func ParseConfig(content string, format ConfigFormat, target interface{}) error {
	switch format {
	case FormatYAML:
		return yaml.Unmarshal([]byte(content), target)
	case FormatJSON:
		return json.Unmarshal([]byte(content), target)
	case FormatTOML:
		return toml.Unmarshal([]byte(content), target)
	case FormatProperties:
		return parseProperties(content, target)
	default:
		return fmt.Errorf("unsupported format: %s", format)
	}
}

// FormatConfig 将对象格式化为指定格式的字符串
func FormatConfig(data interface{}, format ConfigFormat) (string, error) {
	switch format {
	case FormatYAML:
		bytes, err := yaml.Marshal(data)
		return string(bytes), err
	case FormatJSON:
		bytes, err := json.MarshalIndent(data, "", "  ")
		return string(bytes), err
	case FormatTOML:
		var buf strings.Builder
		err := toml.NewEncoder(&buf).Encode(data)
		return buf.String(), err
	case FormatProperties:
		return formatProperties(data)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// parseProperties 解析 properties 格式
func parseProperties(content string, target interface{}) error {
	props := make(map[string]interface{})
	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		props[key] = value
	}

	// 转换为 JSON 再解析到目标对象
	jsonBytes, err := json.Marshal(props)
	if err != nil {
		return err
	}

	return json.Unmarshal(jsonBytes, target)
}

// formatProperties 格式化为 properties 格式
func formatProperties(data interface{}) (string, error) {
	// 先转为 map
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	var props map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &props); err != nil {
		return "", err
	}

	var builder strings.Builder
	for key, value := range props {
		builder.WriteString(fmt.Sprintf("%s=%v\n", key, value))
	}

	return builder.String(), nil
}
