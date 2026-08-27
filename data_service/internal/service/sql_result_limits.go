package service

import "encoding/json"

const (
	developmentSQLMaxRows  = 500
	developmentSQLMaxBytes = 5 * 1024 * 1024
	developmentSQLTimeout  = 30
)

type SQLResultLimits struct {
	MaxRows    int `json:"maxRows"`
	MaxBytes   int `json:"maxBytes"`
	TimeoutSec int `json:"timeoutSeconds"`
}

func developmentSQLLimits() SQLResultLimits {
	return SQLResultLimits{MaxRows: developmentSQLMaxRows, MaxBytes: developmentSQLMaxBytes, TimeoutSec: developmentSQLTimeout}
}

func jsonResultSize(value any) int {
	payload, err := json.Marshal(value)
	if err != nil {
		return developmentSQLMaxBytes + 1
	}
	return len(payload)
}

func admitSQLResultRow(rowCount, currentBytes, maxRows, maxBytes int, row any) (bool, string, int) {
	if rowCount >= maxRows {
		return false, "rows", currentBytes
	}
	rowBytes := jsonResultSize(row)
	if currentBytes+rowBytes > maxBytes {
		return false, "bytes", currentBytes
	}
	return true, "", currentBytes + rowBytes
}
