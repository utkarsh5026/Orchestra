package worker

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type Api struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}

func (a *Api) initializeRouter() {
	a.Router = chi.NewRouter()
	a.Router.Use(middleware.Logger)
	a.Router.Use(middleware.Recoverer)

	a.Router.Route("/tasks", func(r chi.Router) {
		r.Post("/", a.StartTaskHandler)
		r.Get("/", a.GetTasksHandler)
		r.Delete("/{taskID}", a.StopTaskHandler)
	})
}

func (a *Api) Start() {
	a.initializeRouter()

	log.Printf("Starting server on %s:%d", a.Address, a.Port)
	http.ListenAndServe(fmt.Sprintf("%s:%d", a.Address, a.Port), a.Router)
}
