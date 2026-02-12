package services

import (
	"context"
	"encoding/json"
	"fmt"

	"app/internal/models"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

func LogInfo(ctx context.Context, message string, fields map[string]any) {
	runtime.LogInfo(ctx, formatLog(message, fields))
}

func LogError(ctx context.Context, message string, fields map[string]any) {
	runtime.LogError(ctx, formatLog(message, fields))
}

func formatLog(message string, fields map[string]any) string {
	if len(fields) == 0 {
		return message
	}
	b, err := json.Marshal(fields)
	if err != nil {
		return message
	}
	return fmt.Sprintf("%s %s", message, string(b))
}

func RedactedProfileFields(profile models.ConnectionProfile) map[string]any {
	return map[string]any{
		"id":       profile.ID,
		"name":     profile.Name,
		"protocol": profile.Protocol,
		"host":     profile.Host,
		"port":     profile.Port,
		"path":     profile.Path,
	}
}

