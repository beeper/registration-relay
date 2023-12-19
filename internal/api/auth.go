package api

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

var httpClient = &http.Client{}

type authResp struct {
	Identifier string `json:"identifier"`
}

func (a *api) requireAuthHandler(
	validateURL string,
	next http.HandlerFunc,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authToken := r.Header.Get("X-Beeper-Access-Token")

		if authToken == "" {
			a.log.Warn().Msg("Request missing auth header")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		req, err := http.NewRequest(http.MethodGet, validateURL, nil)
		if err != nil {
			a.log.Err(err).Msg("Failed to create request to auth validation URL")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		req.Header.Add("Authorization", authToken)

		resp, err := httpClient.Do(req)
		if err != nil {
			a.log.Err(err).Msg("Failed to make request to auth validation URL")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 500 {
			a.log.Error().
				Int("status_code", resp.StatusCode).
				Msg("Unexpected status from auth URL")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if resp.StatusCode != 200 {
			a.log.Warn().
				Int("status_code", resp.StatusCode).
				Msg("Unauthorized status from auth URL")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		var response authResp
		if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
			a.log.Err(err).Msg("Failed to decode auth response")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		hlog.FromRequest(r).UpdateContext(func(c zerolog.Context) zerolog.Context {
			return c.Str("identifier", response.Identifier)
		})

		next(w, r)
	}
}
