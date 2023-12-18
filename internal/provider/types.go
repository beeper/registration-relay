package provider

import "encoding/json"

type rawCommand struct {
	Command string          `json:"command"`
	Data    json.RawMessage `json:"data"`
}

type registerCommandData struct {
	Code string `json:"code"`
}
