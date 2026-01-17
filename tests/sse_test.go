package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ServerEvent mirrors the generated type for testing
type ServerEvent struct {
	Type string
	Data []byte
	ID   string
}

func (e *ServerEvent) Decode(v any) error {
	return json.Unmarshal(e.Data, v)
}

// EventStream mirrors the generated type for testing
type EventStream struct {
	resp    *http.Response
	scanner *bufio.Scanner
	current *ServerEvent
	err     error
}

func newEventStream(resp *http.Response) *EventStream {
	return &EventStream{
		resp:    resp,
		scanner: bufio.NewScanner(resp.Body),
	}
}

func (s *EventStream) Next() bool {
	if s.err != nil {
		return false
	}

	event := &ServerEvent{}
	var data []byte

	for s.scanner.Scan() {
		line := s.scanner.Bytes()

		if len(line) == 0 {
			if len(data) > 0 {
				event.Data = bytes.TrimSuffix(data, []byte("\n"))
				s.current = event
				return true
			}
			continue
		}

		switch {
		case bytes.HasPrefix(line, []byte("event:")):
			event.Type = string(bytes.TrimSpace(line[6:]))
		case bytes.HasPrefix(line, []byte("data:")):
			data = append(data, bytes.TrimSpace(line[5:])...)
			data = append(data, '\n')
		case bytes.HasPrefix(line, []byte("id:")):
			event.ID = string(bytes.TrimSpace(line[3:]))
		}
	}

	if len(data) > 0 {
		event.Data = bytes.TrimSuffix(data, []byte("\n"))
		s.current = event
		return true
	}

	s.err = s.scanner.Err()
	return false
}

func (s *EventStream) Current() *ServerEvent { return s.current }
func (s *EventStream) Err() error            { return s.err }
func (s *EventStream) Close() error          { return s.resp.Body.Close() }

// Writer mirrors the generated SSE writer
type Writer struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func NewWriter(w http.ResponseWriter) (*Writer, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, fmt.Errorf("streaming not supported")
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	return &Writer{w: w, flusher: flusher}, nil
}

func (w *Writer) Send(eventType string, data any) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return w.SendRaw(eventType, jsonData)
}

func (w *Writer) SendRaw(eventType string, data []byte) error {
	if eventType != "" {
		if _, err := fmt.Fprintf(w.w, "event: %s\n", eventType); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(w.w, "data: %s\n\n", data); err != nil {
		return err
	}
	w.flusher.Flush()
	return nil
}

func TestEventStreamBasic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse, err := NewWriter(w)
		require.NoError(t, err)

		sse.Send("message", map[string]string{"content": "hello"})
		sse.Send("message", map[string]string{"content": "world"})
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	stream := newEventStream(resp)

	var messages []string
	for stream.Next() {
		require.Equal(t, "message", stream.Current().Type)
		var msg struct{ Content string }
		require.NoError(t, stream.Current().Decode(&msg))
		messages = append(messages, msg.Content)
	}
	require.NoError(t, stream.Err())

	assert.Equal(t, []string{"hello", "world"}, messages)
}

func TestEventStreamWithEventID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")

		flusher := w.(http.Flusher)
		fmt.Fprint(w, "id: evt-1\nevent: update\ndata: {\"value\":1}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "id: evt-2\nevent: update\ndata: {\"value\":2}\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	stream := newEventStream(resp)

	var ids []string
	for stream.Next() {
		ids = append(ids, stream.Current().ID)
	}
	require.NoError(t, stream.Err())

	assert.Equal(t, []string{"evt-1", "evt-2"}, ids)
}

func TestEventStreamMultipleDataLines(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		// Multiple data lines should be concatenated with newlines
		fmt.Fprint(w, "data: line1\ndata: line2\ndata: line3\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	resp, err := http.Get(server.URL)
	require.NoError(t, err)
	defer resp.Body.Close()

	stream := newEventStream(resp)

	require.True(t, stream.Next())
	assert.Equal(t, "line1\nline2\nline3", string(stream.Current().Data))
	require.False(t, stream.Next())
	require.NoError(t, stream.Err())
}

func TestEventStreamContextCancellation(t *testing.T) {
	eventsSent := make(chan int, 100)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sse, _ := NewWriter(w)
		for i := 0; ; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
				sse.Send("", i)
				eventsSent <- i
				time.Sleep(10 * time.Millisecond)
			}
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, server.URL, nil)
	require.NoError(t, err)

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	stream := newEventStream(resp)

	count := 0
	for stream.Next() {
		count++
	}

	// Should have received some but not infinite events
	assert.Greater(t, count, 0)
	assert.Less(t, count, 20)
}

func TestServerEventDecode(t *testing.T) {
	event := &ServerEvent{
		Type: "update",
		Data: []byte(`{"id": 123, "name": "test"}`),
		ID:   "evt-1",
	}

	var data struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}
	require.NoError(t, event.Decode(&data))

	assert.Equal(t, 123, data.ID)
	assert.Equal(t, "test", data.Name)
}

func TestWriterSend(t *testing.T) {
	rec := httptest.NewRecorder()
	writer, err := NewWriter(rec)
	require.NoError(t, err)

	err = writer.Send("message", map[string]string{"text": "hello"})
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, "event: message\n")
	assert.Contains(t, body, `data: {"text":"hello"}`)
}

func TestWriterSendWithoutEventType(t *testing.T) {
	rec := httptest.NewRecorder()
	writer, err := NewWriter(rec)
	require.NoError(t, err)

	err = writer.Send("", map[string]int{"value": 42})
	require.NoError(t, err)

	body := rec.Body.String()
	assert.NotContains(t, body, "event:")
	assert.Contains(t, body, `data: {"value":42}`)
}

func TestWriterSendRaw(t *testing.T) {
	rec := httptest.NewRecorder()
	writer, err := NewWriter(rec)
	require.NoError(t, err)

	err = writer.SendRaw("custom", []byte("raw data"))
	require.NoError(t, err)

	body := rec.Body.String()
	assert.Contains(t, body, "event: custom\n")
	assert.Contains(t, body, "data: raw data\n\n")
}

func TestWriterSetsHeaders(t *testing.T) {
	rec := httptest.NewRecorder()
	_, err := NewWriter(rec)
	require.NoError(t, err)

	assert.Equal(t, "text/event-stream", rec.Header().Get("Content-Type"))
	assert.Equal(t, "no-cache", rec.Header().Get("Cache-Control"))
	assert.Equal(t, "keep-alive", rec.Header().Get("Connection"))
}
