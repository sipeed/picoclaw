package wecom

import (
	"path/filepath"
	"testing"
	"time"
)

func TestReqIDStorePersistsRoutes(t *testing.T) {
	storePath := filepath.Join(t.TempDir(), "reqids.json")
	store := newReqIDStore(storePath)
	if err := store.Put("chat-1", "req-1", 2, time.Hour); err != nil {
		t.Fatalf("Put() error = %v", err)
	}

	reloaded := newReqIDStore(storePath)
	route, ok := reloaded.Get("chat-1")
	if !ok {
		t.Fatal("expected persisted route to be loaded")
	}
	if route.ChatID != "chat-1" || route.ReqID != "req-1" || route.ChatType != 2 {
		t.Fatalf("loaded route = %+v", route)
	}
}
