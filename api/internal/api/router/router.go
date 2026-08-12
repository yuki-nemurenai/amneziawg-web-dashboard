package router

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/api/handler"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/api/middleware"
	"github.com/yuki-nemurenai/amneziawg-web-dashboard/api/internal/service"
)

func NewRouter(awgService service.AWGService, authService service.AuthService, webAssets fs.FS) *chi.Mux {
	r := chi.NewRouter()

	// Middlewares
	r.Use(chimiddleware.Recoverer)
	r.Use(middleware.Logger)

	// CORS configuration
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link", "Content-Disposition"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Handlers
	authH := handler.NewAuthHandler(authService)
	statusH := handler.NewStatusHandler(awgService)
	serverH := handler.NewServerHandler(awgService)
	clientH := handler.NewClientHandler(awgService)

	// REST API Routes
	r.Route("/api", func(api chi.Router) {
		// Public Auth Endpoints
		api.Get("/auth/status", authH.GetStatus)
		api.Post("/auth/setup", authH.Setup)
		api.Post("/auth/login", authH.Login)

		// Protected Routes (Require JWT Auth)
		api.Group(func(protected chi.Router) {
			protected.Use(middleware.AuthMiddleware(authService))

			protected.Get("/auth/me", authH.Me)
			protected.Post("/auth/change-password", authH.ChangePassword)
			protected.Get("/status", statusH.GetStatus)
			protected.Get("/server", serverH.GetServerConfig)
			protected.Post("/server", serverH.UpdateServerConfig)

			protected.Get("/clients", clientH.ListClients)
			protected.Post("/clients", clientH.CreateClient)
			protected.Delete("/clients/{name}", clientH.DeleteClient)
			protected.Get("/clients/{name}/qr", clientH.GetClientQRCode)
			protected.Get("/clients/{name}/download", clientH.DownloadClientConfig)
		})
	})

	// Healthcheck endpoint
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Static Web Frontend Asset Serving (if compiled embedded)
	if webAssets != nil {
		fileServer := http.FileServer(http.FS(webAssets))
		r.Handle("/*", http.StripPrefix("/", fileServer))
	} else {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("AmneziaWG Go Backend API Running"))
		})
	}

	return r
}
