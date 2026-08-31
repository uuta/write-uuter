package captureadapter

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/coder/websocket"
)

type cdpClient struct {
	connection *websocket.Conn
	nextID     int64
	events     map[string]int
}

type cdpRequest struct {
	ID        int64  `json:"id"`
	Method    string `json:"method"`
	Params    any    `json:"params,omitempty"`
	SessionID string `json:"sessionId,omitempty"`
}

type cdpMessage struct {
	ID        int64           `json:"id,omitempty"`
	Method    string          `json:"method,omitempty"`
	SessionID string          `json:"sessionId,omitempty"`
	Result    json.RawMessage `json:"result,omitempty"`
	Error     *struct {
		Code int `json:"code"`
	} `json:"error,omitempty"`
}

func newCDPClient(connection *websocket.Conn) *cdpClient {
	return &cdpClient{connection: connection, events: make(map[string]int)}
}

func (client *cdpClient) call(ctx context.Context, method string, parameters any, sessionID string, destination any) error {
	client.nextID++
	id := client.nextID
	payload, err := json.Marshal(cdpRequest{ID: id, Method: method, Params: parameters, SessionID: sessionID})
	if err != nil {
		return err
	}
	if err := client.connection.Write(ctx, websocket.MessageText, payload); err != nil {
		return err
	}
	for {
		message, err := client.read(ctx)
		if err != nil {
			return err
		}
		if message.ID != id {
			client.recordEvent(message)
			continue
		}
		if message.Error != nil {
			return fmt.Errorf("CDP command failed with code %d", message.Error.Code)
		}
		if destination != nil {
			if len(message.Result) == 0 || json.Unmarshal(message.Result, destination) != nil {
				return fmt.Errorf("invalid CDP result")
			}
		}
		return nil
	}
}

func (client *cdpClient) waitEvent(ctx context.Context, method, sessionID string) error {
	key := eventKey(method, sessionID)
	if client.events[key] > 0 {
		client.events[key]--
		return nil
	}
	for {
		message, err := client.read(ctx)
		if err != nil {
			return err
		}
		if message.ID == 0 && message.Method == method && message.SessionID == sessionID {
			return nil
		}
		client.recordEvent(message)
	}
}

func (client *cdpClient) clearEvent(method, sessionID string) {
	delete(client.events, eventKey(method, sessionID))
}

func (client *cdpClient) read(ctx context.Context) (cdpMessage, error) {
	messageType, payload, err := client.connection.Read(ctx)
	if err != nil {
		return cdpMessage{}, err
	}
	if messageType != websocket.MessageText {
		return cdpMessage{}, fmt.Errorf("unexpected CDP message type")
	}
	var message cdpMessage
	if err := json.Unmarshal(payload, &message); err != nil {
		return cdpMessage{}, err
	}
	return message, nil
}

func (client *cdpClient) recordEvent(message cdpMessage) {
	if message.ID == 0 && message.Method != "" {
		client.events[eventKey(message.Method, message.SessionID)]++
	}
}

func eventKey(method, sessionID string) string {
	return sessionID + "\x00" + method
}
