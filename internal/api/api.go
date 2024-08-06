package api

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"sync"

	"github.com/Mauricio-Carrion/GoReactAMA/internal/store/pgstore"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v5"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

const (
	MessageKindMessageCreated = "message"
)

type apiHandler struct {
	queries *pgstore.Queries
	router 	*chi.Mux
	upgrader websocket.Upgrader
	subscribers map[string]map[*websocket.Conn]context.CancelFunc
	mutex *sync.Mutex
}

type MessageCreated struct {
	ID string `json:"id"`
	Message string `json:"message"`
}

type Message struct {
	Kind string `json:"kind"`
	Value any `json:"value"`
	RoomId string `json:"-"`
}

func (handler apiHandler) notifyClients(message Message) {
	handler.mutex.Lock()
	defer handler.mutex.Unlock()

	subscribers, ok := handler.subscribers[message.RoomId]
	if !ok || len(subscribers) == 0 {
		return
	}

	for conn, cancel := range subscribers {
		if err := conn.WriteJSON(message); err != nil {
			slog.Error("Failed to write message", "error", err)
			cancel()
		}
	}
}

func (handler apiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.router.ServeHTTP(writer, request)
}

func NewHandler(queries *pgstore.Queries) http.Handler {
	api := apiHandler{
		queries: queries,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
		subscribers: make(map[string]map[*websocket.Conn]context.CancelFunc),
		mutex: &sync.Mutex{},
	}

	router := chi.NewRouter()
	router.Use(
		middleware.RequestID, 
		middleware.Recoverer, 
		middleware.Logger,
	)

	router.Use(
		cors.Handler(cors.Options{
    AllowedOrigins:   []string{"https://*", "http://*"},
    AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
    AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
    ExposedHeaders:   []string{"Link"},
    AllowCredentials: false,
    MaxAge:           300, 
  }))

	router.Get("/subscribe/{roomId}", api.handleSubscribeToRoom)

	router.Route("/api", func(router chi.Router) {
		router.Route("/rooms", func(router chi.Router) {
			router.Post("/", api.handleCreateRoom)
			router.Get("/", api.handleGetRooms)

			router.Route("/{roomId}/messages", func(router chi.Router) {
				router.Post("/", api.handleCreateRoomMessage)
				router.Get("/", api.handleGetRoomMessages)

				router.Route("/{messageId}", func(router chi.Router) {
					router.Get("/", api.handleGetRoomMessage)
					router.Patch("/react", api.handleReactMessage)
					router.Delete("/react", api.handleDeleteReactMessage)
					router.Patch("/answer", api.handleMarkAnsweredMessage)
				})
			})
		})
	})

	api.router = router

	return api
}

func (handler apiHandler) handleSubscribeToRoom(writer http.ResponseWriter, request *http.Request) {
		rawRoomId := chi.URLParam(request, "roomId")
		roomId, err := uuid.Parse(rawRoomId)
		if err != nil {
			http.Error(writer, "Room ID is not valid", http.StatusBadRequest)
		}

		_, err = handler.queries.GetRoom(context.Background(), roomId)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows){
				http.Error(writer, "Room not found", http.StatusNotFound)
			}

			http.Error(writer, err.Error(), http.StatusInternalServerError)
			return
		}

		conn, err := handler.upgrader.Upgrade(writer, request, nil)
		if err != nil {
			slog.Warn("Failed to upgrade connection", "error", err)
			http.Error(writer, err.Error(), http.StatusBadRequest)
			return
		}

		defer conn.Close()

		ctx, cancel := context.WithCancel(context.Background())

		handler.mutex.Lock()
		if _, ok := handler.subscribers[rawRoomId]; !ok {
			handler.subscribers[rawRoomId] = make(map[*websocket.Conn]context.CancelFunc)
		}
		slog.Info("Subscribing to room", "roomId", rawRoomId, "client_ip", request.RemoteAddr)
		handler.subscribers[rawRoomId][conn] = cancel
		handler.mutex.Unlock()

		<-ctx.Done()

		handler.mutex.Lock()
		delete(handler.subscribers[rawRoomId], conn)
		handler.mutex.Unlock()
}

func (handler apiHandler) handleCreateRoom(writer http.ResponseWriter, request *http.Request) {
	type requestBody struct {
		Theme string `json:"theme"`
	}

	type responseBody struct {
		Id string `json:"id"`
	}

	var body requestBody
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	roomId, err := handler.queries.InsertRoom(context.Background(), body.Theme)
	if err != nil {
		slog.Error("Failed to create room", "error", err)
		http.Error(writer, "Failed to create room", http.StatusInternalServerError)
		return
	}

	data, _ := json.Marshal(responseBody{Id: roomId.String()})

	writer.Header().Set("Content-Type", "application/json")
	writer.Write(data)
}

func (handler apiHandler) handleGetRooms(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleCreateRoomMessage(writer http.ResponseWriter, request *http.Request) {
	rawRoomId := chi.URLParam(request, "roomId")
	roomId, err := uuid.Parse(rawRoomId)

	if err != nil {
		http.Error(writer, "Room ID is not valid", http.StatusBadRequest)
		return
	}

	type requestBody struct {
		Message string `json:"message"`
	}

	type responseBody struct {
		Id string `json:"id"`
	}

	var body requestBody
	if err := json.NewDecoder(request.Body).Decode(&body); err != nil {
		http.Error(writer, "Invalid request body", http.StatusBadRequest)
		return
	}

	messageId, err := handler.queries.InsertMessage(context.Background(), pgstore.InsertMessageParams{
		RoomID: roomId,
		Message: body.Message,
	})

	if err != nil {
		slog.Error("Failed to create room message", "error", err)
		http.Error(writer, "Failed to create room message", http.StatusInternalServerError)
		return
	}

	data, _ := json.Marshal(responseBody{Id: messageId.String()})

	writer.Header().Set("Content-Type", "application/json")

	go handler.notifyClients(Message{
		Kind: MessageKindMessageCreated,
		RoomId: rawRoomId,
		Value: MessageCreated{
			ID: messageId.String(),
			Message: body.Message,
		},
	})

	writer.Write(data)
}

func (handler apiHandler) handleGetRoomMessages(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleGetRoomMessage(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleReactMessage(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleDeleteReactMessage(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleMarkAnsweredMessage(writer http.ResponseWriter, request *http.Request) {
}