package api

import (
	"net/http"

	"github.com/Mauricio-Carrion/GoReactAMA/internal/store/pgstore"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type apiHandler struct {
	queries *pgstore.Queries
	router 	*chi.Mux
}

func (handler apiHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	handler.router.ServeHTTP(writer, request)
}

func NewHandler(queries *pgstore.Queries) http.Handler {
	api := apiHandler{
		queries: queries,
	}

	router:= chi.NewRouter()
	router.Use(
		middleware.RequestID, 
		middleware.Recoverer, 
		middleware.Logger,
	)

	router.Route("/api", func(r chi.Router) {
		router.Route("/rooms", func(r chi.Router) {
			router.Post("/", api.handleCreateRoom)
			router.Get("/", api.handleGetRooms)

			router.Route("/{roomId}/messages", func(r chi.Router) {
				router.Post("/", api.handleCreateRoomMessage)
				router.Get("/", api.handleGetRoomMessages)

				router.Route("/{messageId}", func(r chi.Router) {
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

func (handler apiHandler) handleCreateRoom(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleGetRooms(writer http.ResponseWriter, request *http.Request) {
}

func (handler apiHandler) handleCreateRoomMessage(writer http.ResponseWriter, request *http.Request) {
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