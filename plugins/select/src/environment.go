package main

import "strings"

func defaultEnvironmentColor(name string) string {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "development", "dev":
		return "blue"
	case "staging", "stage":
		return "orange"
	case "production", "prod":
		return "red"
	default:
		return "neutral"
	}
}

func environmentEmoji(color string) string {
	switch color {
	case "blue":
		return "🔵"
	case "orange":
		return "🟠"
	case "red":
		return "🔴"
	case "green":
		return "🟢"
	case "yellow":
		return "🟡"
	case "purple":
		return "🟣"
	default:
		return "⚪"
	}
}

func validEnvironmentColor(color string) bool {
	switch color {
	case "neutral", "blue", "orange", "red", "green", "yellow", "purple":
		return true
	default:
		return false
	}
}

func defaultEnvironments() map[string]Environment {
	return map[string]Environment{
		"Development": {Color: "blue"},
		"Staging":     {Color: "orange"},
		"Production":  {Color: "red"},
	}
}
