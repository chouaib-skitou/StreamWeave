package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"reflect"
	"testing"
	"time"

	"github.com/segmentio/kafka-go"
)

type retryFactStore struct {
	relayStore
	calls []string
	fail  bool
}

func (s *retryFactStore) ProcessFact(_ context.Context, _ string, payload []byte) error {
	s.calls = append(s.calls, string(payload))
	if s.fail {
		s.fail = false
		return errors.New("temporary database failure")
	}
	return nil
}

func (s *retryFactStore) RecordFactFailure(context.Context, string, []byte, string, time.Time) (bool, int32, error) {
	return false, 1, nil
}

type offsetReader struct {
	cancel              context.CancelFunc
	store               *retryFactStore
	fetched             int
	committed           []int64
	commitFailures      int
	advancedBeforeRetry bool
}

func (r *offsetReader) FetchMessage(_ context.Context) (kafka.Message, error) {
	if r.fetched == 2 {
		r.cancel()
		return kafka.Message{}, context.Canceled
	}
	if r.fetched == 1 && len(r.store.calls) < 2 {
		r.advancedBeforeRetry = true
	}
	r.fetched++
	return kafka.Message{Offset: int64(r.fetched), Value: []byte(string(rune('a' + r.fetched - 1)))}, nil
}

func (r *offsetReader) CommitMessages(_ context.Context, messages ...kafka.Message) error {
	if r.commitFailures > 0 {
		r.commitFailures--
		return errors.New("temporary commit failure")
	}
	r.committed = append(r.committed, messages[0].Offset)
	return nil
}

func TestConsumeRetriesSameOffsetBeforeFetchingNext(t *testing.T) {
	for _, commitFailures := range []int{0, 1} {
		t.Run(string(rune('0'+commitFailures)), func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			store := &retryFactStore{fail: true}
			reader := &offsetReader{cancel: cancel, store: store, commitFailures: commitFailures}
			r := &relay{store: store, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			r.consumeMessages(ctx, "inventory", reader)
			if reader.advancedBeforeRetry || !reflect.DeepEqual(reader.committed, []int64{1, 2}) {
				t.Fatalf("advanced=%v committed=%v calls=%v", reader.advancedBeforeRetry, reader.committed, store.calls)
			}
			if len(store.calls) < 3 || store.calls[0] != "a" || store.calls[1] != "a" {
				t.Fatalf("failed message was not retried: %v", store.calls)
			}
		})
	}
}

func TestDLQBodyPreservesMalformedJSON(t *testing.T) {
	for _, payload := range [][]byte{[]byte(`{"event_id":"evt_1"}`), []byte("invalid-json"), {0xff, 0x00}} {
		body, err := json.Marshal(map[string]any{"payload": dlqBody(payload)})
		if err != nil {
			t.Fatal(err)
		}
		var decoded struct {
			Payload json.RawMessage `json:"payload"`
		}
		if err = json.Unmarshal(body, &decoded); err != nil {
			t.Fatal(err)
		}
		if json.Valid(payload) {
			if string(decoded.Payload) != string(payload) {
				t.Fatalf("changed valid JSON: %s", body)
			}
		} else {
			var encoded string
			if err = json.Unmarshal(decoded.Payload, &encoded); err != nil {
				t.Fatal(err)
			}
			if encoded != base64.StdEncoding.EncodeToString(payload) {
				t.Fatalf("lost original bytes: %s", body)
			}
		}
	}
}
