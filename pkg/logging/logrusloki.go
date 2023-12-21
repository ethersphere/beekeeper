package logging

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/ethersphere/beekeeper/pkg/logging/loki"
	"github.com/sirupsen/logrus"
)

type LokiHook struct {
	hostname     string
	lokiEndpoint string
	httpclient   *http.Client
}

func newLoki(lokiEndpoint string) LokiHook {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}
	return LokiHook{
		hostname:     hostname,
		lokiEndpoint: lokiEndpoint,
		httpclient:   &http.Client{Timeout: 5 * time.Second},
	}
}

func (l LokiHook) Levels() []logrus.Level {
	return []logrus.Level{
		logrus.ErrorLevel,
		logrus.WarnLevel,
		logrus.InfoLevel,
		logrus.DebugLevel,
		logrus.TraceLevel,
	}
}

func (l LokiHook) Fire(entry *logrus.Entry) error {
	stream := loki.NewStream()

	msg, err := entry.Logger.Formatter.Format(entry)
	if err != nil {
		return fmt.Errorf("loki format failed: %s", err.Error())
	}
	stream.AddEntry(entry.Time, string(msg))
	stream.AddLabel("hostname", l.hostname)

	batch := loki.NewBatch()
	batch.AddStream(stream)

	err = l.executeHTTPRequest(batch)
	if err != nil {
		return fmt.Errorf("loki request failed: %s", err.Error())
	}
	return nil
}

func (l LokiHook) executeHTTPRequest(batch *loki.Batch) error {
	data, err := json.Marshal(batch)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", l.lokiEndpoint, bytes.NewReader(data))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := l.httpclient.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("error reading response (%s): %s", resp.Status, err.Error())
		}

		return fmt.Errorf("error posting loki batch (%s): %s", resp.Status, string(data))
	}

	return err
}
