package analytics

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
)

var ConfigURL = ""
var ConfigToken = ""
var client http.Client

var logger = log.With().Str("component", "analytics").Logger()

func trackImplementation(userId string, event string, properties map[string]any) {
	var buf bytes.Buffer
	err := json.NewEncoder(&buf).Encode(map[string]interface{}{
		"userId":     userId,
		"event":      event,
		"properties": properties,
	})
	if err != nil {
		logger.Error().Err(err).Msg("error encoding payload")
		return
	}

	req, err := http.NewRequest(http.MethodPost, ConfigURL, &buf)
	if err != nil {
		logger.Error().Err(err).Msg("error creating request")
		return
	}
	req.SetBasicAuth(ConfigToken, "")
	resp, err := client.Do(req)
	if err != nil {
		logger.Error().Err(err).Msg("error sending request")
		return
	}
	err = resp.Body.Close()
	if err != nil {
		logger.Error().Err(err).Msg("error closing request")
	}

	logger.Info().Str("event", event).Msg("Tracked event")
}

func IsEnabled() bool {
	return len(ConfigToken) > 0
}

func Track(userId string, event string, properties map[string]any) {
	if !IsEnabled() {
		return
	}

	go trackImplementation(userId, event, properties)
}
