package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"miru-api/internal/gogoanime"
	"miru-api/internal/mangadex"
	"miru-api/internal/storage"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/redis/go-redis/v9"
)

type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr: mangadex.GetEnv("VALKEY_ADDR", "localhost:6379"),
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Valkey connection failed: %v", err)
	}

	if err := storage.InitDB(ctx); err != nil {
		log.Printf("Warning: Postgres connection failed: %v", err)
	} else {
		log.Println("Postgres initialized successfully")
	}

	mdxClient := mangadex.NewClient(redisClient)
	if err := mdxClient.InitTags(ctx); err != nil {
		log.Printf("Warning: Failed to initialize tags: %v", err)
	}

	gogoClient := gogoanime.NewClient(redisClient)

	progressRepo := storage.NewProgressRepository()

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "https://miru.jose-rodrigues.info", "https://miru-iota.vercel.app"},
		AllowedMethods:   []string{"GET", "POST", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Content-Type"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{Success: true})
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/search", handleSearch(mdxClient))
		r.Get("/tags", handleGetTags(mdxClient))
		r.Post("/progress", handleProgress(progressRepo))
		r.Post("/chapter/{id}", handleChapter(mdxClient))
		r.Get("/manga/{id}", handleMangaDetails(mdxClient))

		r.Post("/anime/search", handleAnimeSearch(gogoClient))
		r.Get("/anime/recent", handleAnimeRecent(gogoClient))
		r.Get("/anime/popular", handleAnimePopular(gogoClient))
		r.Get("/anime/{id}", handleAnimeDetails(gogoClient))
		r.Get("/anime/{id}/episodes", handleAnimeEpisodes(gogoClient))
		r.Get("/anime/episode/{id}/sources", handleAnimeSources(gogoClient))
		r.Get("/anime/episode/{id}/downloads", handleAnimeDownloads(gogoClient))
	})

	port := mangadex.GetEnv("API_PORT", "8080")
	log.Printf("Starting API server on port %s", port)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	redisClient.Close()
	storage.Close()
}

func handleSearch(client *mangadex.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var params mangadex.SearchRequest
		if err := json.NewDecoder(r.Body).Decode(&params); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Invalid request body",
			})
			return
		}

		result, err := client.SearchManga(r.Context(), params)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    result,
		})
	}
}

func handleGetTags(client *mangadex.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    client.GetAllTags(),
		})
	}
}

type ProgressRequest struct {
	UserID     string `json:"user_id"`
	MangaID    string `json:"manga_id"`
	ChapterID  string `json:"chapter_id"`
	PageNumber int    `json:"page_number"`
}

func handleProgress(repo *storage.ProgressRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req ProgressRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Invalid request body",
			})
			return
		}

		if req.UserID == "" || req.MangaID == "" || req.ChapterID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "user_id, manga_id, and chapter_id are required",
			})
			return
		}

		progress := &storage.ReadProgress{
			UserID:     req.UserID,
			MangaID:    req.MangaID,
			ChapterID:  req.ChapterID,
			PageNumber: req.PageNumber,
		}

		if err := repo.Upsert(r.Context(), progress); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    progress,
		})
	}
}

func handleChapter(client *mangadex.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		chapterID := chi.URLParam(r, "id")
		if chapterID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "chapter id is required",
			})
			return
		}

		result, err := client.GetChapterPages(r.Context(), chapterID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    result,
		})
	}
}

func handleMangaDetails(client *mangadex.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		mangaID := chi.URLParam(r, "id")
		if mangaID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "manga id is required",
			})
			return
		}

		result, err := client.GetMangaDetails(r.Context(), mangaID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   err.Error(),
			})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    result,
		})
	}
}

type AnimeSearchRequest struct {
	Query string `json:"query"`
	Page  int    `json:"page"`
}

func handleAnimeSearch(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		var req AnimeSearchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "Invalid request body"})
			return
		}

		if req.Page < 1 {
			req.Page = 1
		}

		result, err := client.Search(r.Context(), req.Query, req.Page)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}

func handleAnimeRecent(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		pageStr := r.URL.Query().Get("page")
		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		result, err := client.GetRecent(r.Context(), page)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}

func handleAnimePopular(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		pageStr := r.URL.Query().Get("page")
		page := 1
		if pageStr != "" {
			if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
				page = p
			}
		}

		result, err := client.GetPopular(r.Context(), page)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}

func handleAnimeDetails(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		animeID := chi.URLParam(r, "id")
		if animeID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "anime id is required"})
			return
		}

		result, err := client.GetAnimeDetails(r.Context(), animeID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}

func handleAnimeEpisodes(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		animeID := chi.URLParam(r, "id")
		if animeID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "anime id is required"})
			return
		}

		result, err := client.GetEpisodes(r.Context(), animeID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}

func handleAnimeSources(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		episodeID := chi.URLParam(r, "id")
		if episodeID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "episode id is required"})
			return
		}

		result, err := client.GetStreamingLinks(r.Context(), episodeID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}

func handleAnimeDownloads(client *gogoanime.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		episodeID := chi.URLParam(r, "id")
		if episodeID == "" {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: "episode id is required"})
			return
		}

		result, err := client.GetDownloadLinks(r.Context(), episodeID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(APIResponse{Success: false, Error: err.Error()})
			return
		}

		json.NewEncoder(w).Encode(APIResponse{Success: true, Data: result})
	}
}
