package main

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	uuid "github.com/satori/go.uuid"
)

const (
	writeWait      = 10 * time.Second
	handleWait     = 60 * time.Second
	pingPeriod     = 30 * time.Second
	maxMessageSize = 1024
	endpoint       = "wss://events.ocean.one"
)

type BlazeMessage struct {
	Id     string                 `json:"id"`
	Action string                 `json:"action"`
	Params map[string]interface{} `json:"params,omitempty"`
	Data   interface{}            `json:"data,omitempty"`
	Error  string                 `json:"error,omitempty"`
}

type MessageHandler interface {
	OnOrderMessage(*BlazeMessage) error
}

type Client struct {
	receive chan *BlazeMessage
	handler MessageHandler
	base    string
	quote   string
}

func NewClient(ctx context.Context, base, quote string, h MessageHandler) *Client {
	return &Client{
		receive: make(chan *BlazeMessage, 0),
		handler: h,
		base:    base,
		quote:   quote,
	}
}

func (client *Client) PollOceanMessage(ctx context.Context) error {
	for {
		ctx, cancel := context.WithCancel(ctx)
		dialer := websocket.DefaultDialer
		conn, _, err := dialer.Dial(endpoint, nil)
		if err != nil {
			continue
		}
		if err := client.Subscribe(ctx, conn); err != nil {
			continue
		}

		go client.WritePump(ctx, conn, []byte("ping"))
		go client.ReadPump(ctx, conn)
		if err := client.process(ctx); err != nil {
			cancel()
			conn.Close()
			log.Println(err)
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (client *Client) process(ctx context.Context) error {
	for {
		select {
		case msg := <-client.receive:
			if err := client.handler.OnOrderMessage(msg); err != nil {
				if strings.Contains(err.Error(), WrongSequenceError) {
					return err
				}
			}
		case <-time.After(handleWait):
			return errors.New("no message in 60s, reconnecting...")
		}
	}
}

func (client *Client) Subscribe(ctx context.Context, conn *websocket.Conn) error {
	msg := BlazeMessage{
		Id:     uuid.Must(uuid.NewV4()).String(),
		Action: "SUBSCRIBE_BOOK",
		Params: map[string]interface{}{
			"market": client.base + "-" + client.quote,
		},
	}
	bt, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	return WriteGzipToConn(ctx, conn, bt)
}

func (client *Client) ParseMessage(ctx context.Context, wsReader io.Reader) error {
	var message BlazeMessage
	gzReader, err := gzip.NewReader(wsReader)
	if err != nil {
		return err
	}
	defer gzReader.Close()
	if err = json.NewDecoder(gzReader).Decode(&message); err != nil {
		return err
	}
	if message.Action != "EMIT_EVENT" {
		return nil
	}

	select {
	case client.receive <- &message:
	case <-time.After(writeWait):
		return errors.New("timeout to pipe receive message")
	}
	return nil
}

func (client *Client) WritePump(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	defer conn.Close()

	pingTicker := time.NewTicker(pingPeriod)
	defer pingTicker.Stop()

	for {
		select {
		case <-pingTicker.C:
			err := conn.WriteMessage(websocket.PingMessage, msg)
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

func (client *Client) ReadPump(ctx context.Context, conn *websocket.Conn) error {
	defer conn.Close()

	for {
		messageType, wsReader, err := conn.NextReader()
		if err != nil {
			return err
		}
		if messageType == websocket.BinaryMessage {
			client.ParseMessage(ctx, wsReader)
		}
	}
}

func WriteGzipToConn(ctx context.Context, conn *websocket.Conn, msg []byte) error {
	wsWriter, err := conn.NextWriter(websocket.BinaryMessage)
	if err != nil {
		return err
	}
	gzWriter, err := gzip.NewWriterLevel(wsWriter, 3)
	if err != nil {
		return err
	}

	if _, err := gzWriter.Write(msg); err != nil {
		return err
	}

	if err := gzWriter.Close(); err != nil {
		return err
	}
	if err := wsWriter.Close(); err != nil {
		return err
	}
	return nil
}
