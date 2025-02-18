package websocket

import (
	"context"

	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/gorilla/websocket"
)

// ListenWebSocket listens for messages on a websocket connection.
func ListenWebSocket(ctx context.Context, ws *websocket.Conn, logger logging.Logger) (<-chan string, func(), error) {
	ch := make(chan string)
	readCh := make(chan []byte)
	errCh := make(chan error)
	done := make(chan struct{})

	go func() {
		defer close(readCh)
		defer close(errCh)

		for {
			_, data, err := ws.ReadMessage()
			if err != nil {
				errCh <- err
				return
			}
			select {
			case readCh <- data:
			case <-done:
				return
			}
		}
	}()

	go func() {
		defer close(ch)

		for {
			select {
			case data := <-readCh:
				select {
				case ch <- string(data):
				case <-done:
					return
				}
			case err := <-errCh:
				logger.Infof("websocket error: %v", err)
				return
			case <-ctx.Done():
				logger.Info("context canceled, closing websocket")
				return
			case <-done:
				return
			}
		}
	}()

	closer := func() {
		close(done)
		ws.Close()
	}

	return ch, closer, nil
}
