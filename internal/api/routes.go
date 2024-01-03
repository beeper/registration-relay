package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/beeper/registration-relay/internal/provider"
)

var upgrader = websocket.Upgrader{}

func (a *api) bridgeExecuteCommand(w http.ResponseWriter, r *http.Request) {
	code, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	log := a.log.With().Str("code", code).Logger()

	provider, exists := provider.GetProvider(code)
	if !exists {
		log.Warn().Msg("No provider found for code")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	command := chi.URLParam(r, "command")

	resp, err := provider.ExecuteCommand(command)
	if err != nil {
		log.Err(err).Msg("Failed to execute command")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func (a *api) providerWebsocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		a.log.Err(err).Msg("Failed to upgrade websocket connection")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer conn.Close()

	provider := provider.NewProvider(conn, a.secret)
	provider.WebsocketLoop()

	a.log.Info().Msg("Websocket connection closed")
}
