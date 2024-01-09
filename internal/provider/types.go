package provider

type RawCommand[T any] struct {
	Command string `json:"command"`
	ReqID   int    `json:"id"`
	Data    T      `json:"data"`
}

type versions struct {
	HardwareVersion string `json:"hardware_version"`
}

type registerCommandData struct {
	Code     string   `json:"code"`
	Secret   string   `json:"secret"`
	Commit   string   `json:"commit,omitempty"`
	Versions versions `json:"versions,omitempty"`
}

type errorData struct {
	Error string `json:"error"`
}
