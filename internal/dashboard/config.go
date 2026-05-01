package dashboard

import (
	"os"
	"strconv"
	"strings"
)

const (
	defaultHistoryLimit = 1000
	defaultPayloadLimit = 16 * 1024

	historyLimitEnv = "MA_DASHBOARD_HISTORY_LIMIT"
	recentLimitEnv  = "MA_DASHBOARD_RECENT_LIMIT"
	payloadLimitEnv = "MA_DASHBOARD_PAYLOAD_LIMIT"
)

func historyLimit() int {
	return positiveIntEnv(historyLimitEnv, defaultHistoryLimit)
}

func recentRunLimit() int {
	return positiveIntEnv(recentLimitEnv, defaultRecentRunLimit)
}

func payloadLimit() int {
	return positiveIntEnv(payloadLimitEnv, defaultPayloadLimit)
}

func positiveIntEnv(key string, fallback int) int {
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
