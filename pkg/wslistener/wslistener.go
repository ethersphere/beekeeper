package wslistener

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/ethersphere/beekeeper/pkg/bee"
	"github.com/ethersphere/beekeeper/pkg/logging"
	"github.com/gorilla/websocket"
)

// ListenWebSocket listens for messages on a websocket connection.
func ListenWebSocket(ctx context.Context, client *bee.Client, endpoint string, logger logging.Logger) (<-chan string, func(), error) {
	dialer := &websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 45 * time.Second,
	}

	ws, _, err := dialer.DialContext(ctx, fmt.Sprintf("ws://%s%s", client.Host(), endpoint), http.Header{})
	if err != nil {
		return nil, nil, err
	}

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
			case <-ctx.Done():
				return
			}
		}
	}()

	go func() {
		defer close(ch)

		for {
			select {
			case data := <-readCh:
				logger.WithField("node", client.Name()).Infof("websocket received message: %s", string(data))
				select {
				case ch <- string(data):
				case <-done:
					return
				}
			case err := <-errCh:
				logger.Errorf("websocket error: %v", err)
				return
			case <-ctx.Done():
				logger.Info("context canceled, closing websocket")
				return
			case <-done:
				return
			}
		}
	}()

	var once sync.Once
	closer := func() {
		once.Do(func() {
			deadline := time.Now().Add(5 * time.Second)
			msg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			if err := ws.WriteControl(websocket.CloseMessage, msg, deadline); err != nil {
				logger.Errorf("failed to send close message: %v", err)
			}

			close(done)
			if err := ws.Close(); err != nil {
				logger.Errorf("failed to close websocket: %v", err)
			}
		})
	}

	return ch, closer, nil
}
