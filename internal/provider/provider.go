package provider

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/registration-relay/internal/util"
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

func GetProvider(code string) (*provider, bool) {
	codeToProviderLock.Lock()
	defer codeToProviderLock.Unlock()
	p, exists := codeToProvider[code]
	return p, exists
}

func RegisterProvider(code string, provider *provider) (string, error) {
	codeToProviderLock.Lock()
	defer codeToProviderLock.Unlock()

	if existing, exists := codeToProvider[code]; exists {
		existing.log.Warn().
			Str("code", code).
			Msg("New provider with same code registering, exiting websocket")
		existing.ws.Close()
	}

	if code == "" {
		var err error
		code, err = util.GenerateProviderCode()
		if err != nil {
			return "", err
		}
	}

	codeToProvider[code] = provider
	return code, nil
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
	reqID      int
}

func NewProvider(ws *websocket.Conn) *provider {
	logger := log.With().
		Str("component", "provider").
		Logger()

	return &provider{
		log:       logger,
		ws:        ws,
		resultsCh: make(chan json.RawMessage),
		reqID:     1,
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

		var rawCommand RawCommand[json.RawMessage]
		if err := json.Unmarshal(message, &rawCommand); err != nil {
			p.log.Err(err).Msg("Failed to decode websocket message")
			break
		}

		switch rawCommand.Command {
		case "register":
			// Intercept and handle register command here
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
			buf, err := json.Marshal(RawCommand[registerCommandData]{Command: "response", Data: response, ReqID: rawCommand.ReqID})
			if err != nil {
				p.log.Err(err).Msg("Failed to encode register response")
				break
			}
			p.ws.WriteMessage(websocket.TextMessage, buf)

			// Set registered flag, enabling commands from bridge to come in
			p.registered = true
		case "ping":
			buf, err := json.Marshal(RawCommand[struct{}]{Command: "pong", ReqID: rawCommand.ReqID})
			if err != nil {
				p.log.Err(err).Msg("Failed to encode ping response")
				break
			}
			p.ws.WriteMessage(websocket.TextMessage, buf)
		case "response":
			// Otherwise pass to the results channel, we're expecting a listener
			select {
			case p.resultsCh <- rawCommand.Data:
			default:
				p.log.Warn().Msg("Failed to send response, no request waiter!")
			}
		default:
			p.log.Warn().Str("command", rawCommand.Command).Msg("Received unknown command")
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

	p.reqID++

	cmd := RawCommand[json.RawMessage]{Command: command, ReqID: p.reqID}
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
