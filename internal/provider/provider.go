package provider

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/validation-relay/internal/util"
)

// Map codes -> providers, management
//

var (
	codeToProvider     map[string]*provider
	codeToProviderLock sync.Mutex
)

func init() {
	codeToProvider = make(map[string]*provider, 0)
}

func GetProvider(key string) (*provider, bool) {
	codeToProviderLock.Lock()
	defer codeToProviderLock.Unlock()
	p, exists := codeToProvider[key]
	return p, exists
}

func RegisterProvider(key string, provider *provider) (string, error) {
	codeToProviderLock.Lock()
	defer codeToProviderLock.Unlock()

	if key == "" {
		var err error
		key, err = util.GenerateProviderCode()
		if err != nil {
			return "", err
		}
	}

	codeToProvider[key] = provider
	return key, nil
}

func UnregisterProvider(key string) {
	codeToProviderLock.Lock()
	defer codeToProviderLock.Unlock()
	delete(codeToProvider, key)
}

// Actual provider implementation
//

type provider struct {
	log        zerolog.Logger
	cmdLock    sync.Mutex
	registered bool
	ws         *websocket.Conn
	resultsCh  chan json.RawMessage
}

func NewProvider(ws *websocket.Conn) *provider {
	logger := log.With().
		Str("component", "provider").
		Logger()

	return &provider{
		log:       logger,
		ws:        ws,
		resultsCh: make(chan json.RawMessage),
	}
}

func (p *provider) WebsocketLoop() {
	registerCode := ""

	for {
		_, message, err := p.ws.ReadMessage()
		if err != nil {
			p.log.Err(err).Msg("Websocket read error")
			break
		}

		var rawCommand rawCommand
		if err := json.Unmarshal(message, &rawCommand); err != nil {
			p.log.Err(err).Msg("Failed to decode websocket message")
			break
		}

		// Intercept and handle register command here
		if rawCommand.Command == "register" {
			var request registerCommandData
			if err := json.Unmarshal(rawCommand.Data, &request); err != nil {
				p.log.Err(err).Msg("Failed to decode register request")
				break
			}
			registerCode, err = RegisterProvider(request.Code, p)
			if err != nil {
				p.log.Err(err).Msg("Failed to register provider")
				break
			}

			// Send back register response before setting the flag (ws is single writer)
			response := registerCommandData{registerCode}
			buf, err := json.Marshal(response)
			if err != nil {
				p.log.Err(err).Msg("Failed to encode register response")
				break
			}
			p.ws.WriteMessage(websocket.TextMessage, buf)

			// Set registered flag, enabling commands from bridge to come in
			p.registered = true
			continue
		}

		// Otherwise pass to the results channel, we're expecting a listener
		select {
		case p.resultsCh <- rawCommand.Data:
		default:
			p.log.Warn().Msg("Failed to send response, no request waiter!")
		}
	}

	p.log.Info().Msg("Exit provider websocket loop")
	if registerCode != "" {
		UnregisterProvider(registerCode)
	}
}

func (p *provider) ExecuteCommand(command string) (json.RawMessage, error) {
	if !p.registered {
		return nil, nil
	}

	p.cmdLock.Lock()
	defer p.cmdLock.Unlock()

	cmd := rawCommand{Command: command}
	buf, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	// Send over our command and listen for the result
	go p.ws.WriteMessage(websocket.TextMessage, buf)
	result := <-p.resultsCh

	p.log.Info().Bytes("TEST", result).Msg("")

	return result, nil
}
