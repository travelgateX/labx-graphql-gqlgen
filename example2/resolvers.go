//go:generate go run ../../testdata/gqlgen.go

package example2

import (
	context "context"
	"math/rand"
	"sync"
	"time"
	"log"
)

type resolver struct {
	Rooms map[string]*Chatroom
	mu    sync.Mutex // nolint: structcheck
}

func (r *resolver) Mutation() MutationResolver {
	return &mutationResolver{r}
}

func (r *resolver) Query() QueryResolver {
	return &queryResolver{r}
}

func (r *resolver) Subscription() SubscriptionResolver {
	return &subscriptionResolver{r}
}

func New() Config {
	return Config{
		Resolvers: &resolver{
			Rooms: map[string]*Chatroom{},
		},
	}
}

type Chatroom struct {
	Name      string
	Messages  []Message
	Observers map[string]chan *Message
}

type mutationResolver struct{ *resolver }

func (r *mutationResolver) Post(ctx context.Context, text string, username string, roomName string) (*Message, error) {
	log.Printf("Post message [%v] to room: %v", text, roomName)
	r.mu.Lock()
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{Name: roomName, Observers: map[string]chan *Message{}}
		r.Rooms[roomName] = room
	}
	r.mu.Unlock()

	message := Message{
		ID:        randString(8),
		CreatedAt: time.Now(),
		Text:      text,
		CreatedBy: username,
	}

	room.Messages = append(room.Messages, message)
	r.mu.Lock()
	for _, observer := range room.Observers {
		observer <- &message
		log.Printf("observer message length: %v", len(observer))
	}
	r.mu.Unlock()
	return &message, nil
}

type queryResolver struct{ *resolver }

func (r *queryResolver) Room(ctx context.Context, name string) (*Chatroom, error) {
	r.mu.Lock()
	room := r.Rooms[name]
	if room == nil {
		room = &Chatroom{Name: name, Observers: map[string]chan *Message{}}
		r.Rooms[name] = room
	}
	r.mu.Unlock()

	return room, nil
}

type subscriptionResolver struct{ *resolver }

func (r *subscriptionResolver) MessageAdded(ctx context.Context, roomName string) (<-chan *Message, error) {
	log.Printf("susbcription to %v", roomName)
	r.mu.Lock()
	room := r.Rooms[roomName]
	if room == nil {
		room = &Chatroom{Name: roomName, Observers: map[string]chan *Message{}}
		r.Rooms[roomName] = room
	}
	r.mu.Unlock()

	id := randString(8)
	events := make(chan *Message, 1)

	go func() {
		log.Printf("go func subscription id: %v ", id)
		<-ctx.Done()
		r.mu.Lock()
		delete(room.Observers, id)
		log.Printf("go func delete id: %v ", id)
		r.mu.Unlock()
	}()

	r.mu.Lock()
	log.Printf("subscription events id: %v, events: %v ", id, events)
	room.Observers[id] = events
	r.mu.Unlock()

	return events, nil
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}
