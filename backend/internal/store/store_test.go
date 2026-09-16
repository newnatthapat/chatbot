package store_test

import (
	"context"
	"errors"
	"testing"

	"chatbot-backend/internal/store"
)

func openTestStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func TestCreateConversationMakesItListable(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, err := s.CreateConversation(ctx)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	convs, err := s.ListConversations(ctx)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}

	if len(convs) != 1 || convs[0].ID != conv.ID {
		t.Fatalf("ListConversations = %+v, want a single conversation with ID %d", convs, conv.ID)
	}
}

func TestListConversationsOrdersNewestFirst(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	first, err := s.CreateConversation(ctx)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	second, err := s.CreateConversation(ctx)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	convs, err := s.ListConversations(ctx)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}

	if len(convs) != 2 || convs[0].ID != second.ID || convs[1].ID != first.ID {
		t.Fatalf("ListConversations = %+v, want newest (%d) before oldest (%d)", convs, second.ID, first.ID)
	}
}

func TestAppendMessageMakesItListable(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, err := s.CreateConversation(ctx)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}

	if _, err := s.AppendMessage(ctx, conv.ID, "user", "hello"); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if _, err := s.AppendMessage(ctx, conv.ID, "assistant", "hi there"); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	msgs, err := s.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("ListMessages returned %d messages, want 2", len(msgs))
	}
	if msgs[0].Role != "user" || msgs[0].Content != "hello" {
		t.Errorf("msgs[0] = %+v, want role=user content=hello", msgs[0])
	}
	if msgs[1].Role != "assistant" || msgs[1].Content != "hi there" {
		t.Errorf("msgs[1] = %+v, want role=assistant content=hi there", msgs[1])
	}
}

func TestGetConversationReturnsErrNotFoundForMissingID(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	_, err := s.GetConversation(ctx, 999)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetConversation err = %v, want store.ErrNotFound", err)
	}
}

func TestDeleteConversationReturnsErrNotFoundForMissingID(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	err := s.DeleteConversation(ctx, 999)
	if !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("DeleteConversation err = %v, want store.ErrNotFound", err)
	}
}

func TestDeleteConversationRemovesItAndItsMessages(t *testing.T) {
	s := openTestStore(t)
	ctx := context.Background()

	conv, err := s.CreateConversation(ctx)
	if err != nil {
		t.Fatalf("CreateConversation: %v", err)
	}
	if _, err := s.AppendMessage(ctx, conv.ID, "user", "hello"); err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}

	if err := s.DeleteConversation(ctx, conv.ID); err != nil {
		t.Fatalf("DeleteConversation: %v", err)
	}

	convs, err := s.ListConversations(ctx)
	if err != nil {
		t.Fatalf("ListConversations: %v", err)
	}
	if len(convs) != 0 {
		t.Fatalf("ListConversations = %+v, want empty after delete", convs)
	}

	msgs, err := s.ListMessages(ctx, conv.ID)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(msgs) != 0 {
		t.Fatalf("ListMessages = %+v, want empty after conversation delete", msgs)
	}
}
