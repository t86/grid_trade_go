package bus

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPublishDeliversEventToSubscriber(t *testing.T) {
	b := New()
	ch := b.Subscribe("orders", 1)

	b.Publish(Event{Topic: "orders", Payload: "ok"})

	select {
	case event := <-ch:
		require.Equal(t, "orders", event.Topic)
		require.Equal(t, "ok", event.Payload)
	case <-time.After(time.Second):
		t.Fatal("expected event to be delivered")
	}
}
