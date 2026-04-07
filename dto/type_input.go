package dto

import (
	"encoding/json"
)

type TypeInput struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}
