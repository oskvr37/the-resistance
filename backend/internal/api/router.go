package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func NewRouter(s *server) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
			"http://localhost",
			"http://127.0.0.1",
		},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "DELETE", "PATCH", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type", "Authorization"},
	}))

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Post("/auth", s.postAuthHandler)

	r.Group(func(r chi.Router) {
		r.Use(s.authMiddleware(s.Config.JWTSecret))

		r.Get("/auth/me", s.meAuthHandler)

		r.Route("/rooms", func(r chi.Router) {
			r.Post("/", s.createRoomHandler)
			r.Post("/join_code/{code}", s.joinRoomByCodeHandler)

			r.Route("/{id}", func(r chi.Router) {
				r.Get("/events", s.roomEventsHandler)
				r.Post("/join", s.joinRoomHandler)
				r.Post("/leave", s.leaveRoomHandler)
				r.Delete("/member/{member}", s.kickUserRoomHandler)
				r.Patch("/settings", s.settingsRoomHandler)
				r.Post("/start", s.doStartGameHandler)

				r.Post("/game/nominate", s.gameNominateHandler)
				r.Post("/game/vote", s.gameVoteHandler)
				r.Post("/game/mission", s.gameMissionHandler)
			})
		})
	})

	return r
}
