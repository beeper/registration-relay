package api

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/gorilla/websocket"

	"github.com/beeper/validation-relay/internal/provider"
)

var upgrader = websocket.Upgrader{}

func (a *api) bridgeExecuteCommand(w http.ResponseWriter, r *http.Request) {
	providerCode, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	provider, exists := provider.GetProvider(providerCode)
	if !exists {
		a.log.Warn().Str("provider_code", providerCode).Msg("No provider found for code")
		w.WriteHeader(http.StatusNotFound)
		return
	}

	command := chi.URLParam(r, "command")

	resp, err := provider.ExecuteCommand(command)
	if err != nil {
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

	provider := provider.NewProvider(conn)
	provider.WebsocketLoop()

	a.log.Info().Msg("Websocket connection closed")
}
