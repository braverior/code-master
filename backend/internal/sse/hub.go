package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

type Event struct {
	ID   int64       `json:"id"`
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

type subscriber struct {
	ch chan Event
}

type Hub struct {
	mu          sync.RWMutex
	subscribers map[int64][]*subscriber // taskID -> subscribers
	rdb         *redis.Client
}

func NewHub(rdb *redis.Client) *Hub {
	return &Hub{
		subscribers: make(map[int64][]*subscriber),
		rdb:         rdb,
	}
}

func (h *Hub) Subscribe(taskID int64) (<-chan Event, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sub := &subscriber{ch: make(chan Event, 256)}
	h.subscribers[taskID] = append(h.subscribers[taskID], sub)

	unsub := func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		subs := h.subscribers[taskID]
		for i, s := range subs {
			if s == sub {
				h.subscribers[taskID] = append(subs[:i], subs[i+1:]...)
				close(sub.ch)
				break
			}
		}
		if len(h.subscribers[taskID]) == 0 {
			delete(h.subscribers, taskID)
		}
	}
	return sub.ch, unsub
}

func (h *Hub) Broadcast(taskID int64, event Event) {
	ctx := context.Background()
	key := fmt.Sprintf("codegen:stream:%d", taskID)

	data, _ := json.Marshal(event)
	h.rdb.RPush(ctx, key, string(data))

	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, sub := range h.subscribers[taskID] {
		select {
		case sub.ch <- event:
		default:
			// drop if full
		}
	}
}

func (h *Hub) ReplayFrom(taskID int64, fromID int64) ([]Event, error) {
	ctx := context.Background()
	key := fmt.Sprintf("codegen:stream:%d", taskID)

	items, err := h.rdb.LRange(ctx, key, fromID, -1).Result()
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(items))
	for i, item := range items {
		var ev Event
		if err := json.Unmarshal([]byte(item), &ev); err != nil {
			continue
		}
		ev.ID = fromID + int64(i)
		events = append(events, ev)
	}
	return events, nil
}

func (h *Hub) SetExpire(taskID int64, ttl time.Duration) {
	ctx := context.Background()
	key := fmt.Sprintf("codegen:stream:%d", taskID)
	h.rdb.Expire(ctx, key, ttl)
}

func (h *Hub) GetTotalEvents(taskID int64) int64 {
	ctx := context.Background()
	key := fmt.Sprintf("codegen:stream:%d", taskID)
	count, _ := h.rdb.LLen(ctx, key).Result()
	return count
}

func (h *Hub) GetEventsPage(taskID int64, offset, limit int64) ([]Event, error) {
	ctx := context.Background()
	key := fmt.Sprintf("codegen:stream:%d", taskID)

	end := offset + limit - 1
	items, err := h.rdb.LRange(ctx, key, offset, end).Result()
	if err != nil {
		return nil, err
	}

	events := make([]Event, 0, len(items))
	for i, item := range items {
		var ev Event
		if err := json.Unmarshal([]byte(item), &ev); err != nil {
			continue
		}
		ev.ID = offset + int64(i)
		events = append(events, ev)
	}
	return events, nil
}

func ParseLastEventID(header string) int64 {
	if header == "" {
		return 0
	}
	id, _ := strconv.ParseInt(header, 10, 64)
	return id
}
