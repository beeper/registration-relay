package provider

type RawCommand[T any] struct {
	Command string `json:"command"`
	ReqID   int    `json:"id"`
	Data    T      `json:"data"`
}

type registerCommandData struct {
	Code   string `json:"code"`
	Secret string `json:"secret"`
}

type errorData struct {
	Error string `json:"error"`
}
