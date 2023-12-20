package provider

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/beeper/registration-relay/internal/metrics"
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

func calculateSecret(globalSecret []byte, code string) []byte {
	h := hmac.New(sha256.New, globalSecret)
	h.Write([]byte(code))
	return h.Sum(nil)
}

func RegisterProvider(data registerCommandData, provider *provider) (*registerCommandData, error) {
	codeToProviderLock.Lock()
	defer codeToProviderLock.Unlock()

	if data.Code == "" {
		var err error
		data.Code, err = util.GenerateProviderCode()
		if err != nil {
			return nil, err
		}
		data.Secret = base64.RawStdEncoding.EncodeToString(calculateSecret(provider.globalSecret, data.Code))
	} else {
		if len(data.Code) != 19 || len(data.Secret) > 64 {
			return nil, fmt.Errorf("invalid secret")
		}
		decodedSecret, err := base64.RawStdEncoding.DecodeString(data.Secret)
		if err != nil || !hmac.Equal(calculateSecret(provider.globalSecret, data.Code), decodedSecret) {
			return nil, fmt.Errorf("invalid secret")
		}
		if existing, exists := codeToProvider[data.Code]; exists {
			existing.log.Warn().
				Str("code", data.Code).
				Msg("New provider with same code registering, exiting websocket")
			existing.ws.Close()
		}
	}

	codeToProvider[data.Code] = provider
	return &data, nil
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

	globalSecret []byte
}

func NewProvider(ws *websocket.Conn, secret []byte) *provider {
	logger := log.With().
		Str("component", "provider").
		Logger()

	return &provider{
		log:          logger,
		ws:           ws,
		resultsCh:    make(chan json.RawMessage),
		reqID:        1,
		globalSecret: secret,
	}
}

func (p *provider) WebsocketLoop() {
	metrics.ProviderWebsockets.Inc()
	defer metrics.ProviderWebsockets.Dec()

	registerCode := ""

Loop:
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
				break Loop
			}
			response, err := RegisterProvider(request, p)
			if err != nil {
				p.log.Err(err).Msg("Failed to register provider")
				buf, err := json.Marshal(RawCommand[errorData]{Command: "response", Data: errorData{"invalid token"}, ReqID: rawCommand.ReqID})
				if err == nil {
					p.ws.WriteMessage(websocket.TextMessage, buf)
				}
				break Loop
			}
			p.log.Debug().Msg("Registered provider")

			// Send back register response before setting the flag (ws is single writer)
			buf, err := json.Marshal(RawCommand[registerCommandData]{Command: "response", Data: *response, ReqID: rawCommand.ReqID})
			if err != nil {
				p.log.Err(err).Msg("Failed to encode register response")
				break Loop
			}
			p.ws.WriteMessage(websocket.TextMessage, buf)

			// Set registered flag, enabling commands from bridge to come in
			p.registered = true
		case "ping":
			buf, err := json.Marshal(RawCommand[struct{}]{Command: "pong", ReqID: rawCommand.ReqID})
			if err != nil {
				p.log.Err(err).Msg("Failed to encode ping response")
				break Loop
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
		p.log.Debug().Msg("Unregistered provider")
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
