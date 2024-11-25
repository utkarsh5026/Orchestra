package worker

import "github.com/go-chi/chi/v5"

type Api struct {
	Address string
	Port    int
	Worker  *Worker
	Router  *chi.Mux
}
