package main

import (
	"context"
	"database/sql"
	_ "embed"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/render"

	"spreadlove/db"
	"spreadlove/internal/database"
)

type App struct {
	db      *sql.DB
	queries *db.Queries
}

type MessageResponse struct {
	ID      int64  `json:"id"`
	Content string `json:"content"`
}

type PendingMessageResponse struct {
	ID        int64     `json:"id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
}

func main() {
	app := &App{}

	if err := app.setupDB(); err != nil {
		log.Fatal("Failed to setup database:", err)
	}

	defer app.db.Close()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Set a timeout value on the request context (ctx), that will signal
	// through ctx.Done() that the request has timed out and further
	// processing should be stopped.
	r.Use(middleware.Timeout(60 * time.Second))

	r.Route("/api", func(r chi.Router) {
		r.Get("/messages", app.handleGetRandomMessages)
		r.Post("/message", app.handleSubmitMessage)

		r.Route("/admin", func(r chi.Router) {
			r.Use(basicAuth)
			r.Get("/pending", app.handleGetPendingMessages)
			r.Post("/approve/{id}", app.handleApproveMessage)
			r.Post("/reject/{id}", app.handleRejectMessage)
		})
	})

	server := &http.Server{
		Addr:    ":3000",
		Handler: r,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// Block
	<-quit
	log.Println("Shutdown Server ...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Println("Server Shutdown:", err)
	}

	log.Println("Server exiting")
}

func (app *App) setupDB() error {
	database, queries, err := database.Setup("./love.db")
	if err != nil {
		return err
	}

	app.db = database
	app.queries = queries

	return nil
}

func basicAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		validUser := os.Getenv("ADMIN_USER")
		validPass := os.Getenv("ADMIN_PASSWORD")

		if validUser == "" || validPass == "" {
			log.Fatal("ADMIN_USER and ADMIN_PASSWORD must be set")
		}

		if !ok || username != validUser || password != validPass {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (app *App) handleGetRandomMessages(w http.ResponseWriter, r *http.Request) {
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err != nil {
		limit = 10
	}
	limit = max(1, min(limit, 50))

	msgs, err := app.queries.ListRandomMessages(r.Context(), int64(limit))
	if err != nil {
		log.Printf("Error fetching random messages: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := make([]MessageResponse, len(msgs))
	for i, msg := range msgs {
		response[i] = MessageResponse{ID: msg.ID, Content: msg.Content}
	}

	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, proxy-revalidate")

	render.JSON(w, r, response)
}

func (app *App) handleSubmitMessage(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusBadRequest)
		return
	}
	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Message content cannot be empty", http.StatusBadRequest)
		return
	}

	if len(content) > 5000 {
		http.Error(w, "Message too long", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	_, err := app.queries.CreatePendingMessage(ctx, content)

	if err != nil {
		log.Printf("Error submitting message message: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (app *App) handleGetPendingMessages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	msgs, err := app.queries.ListPendingMessages(ctx)

	if err != nil {
		log.Printf("Error fetching pending messages: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	response := make([]PendingMessageResponse, len(msgs))

	for i, msg := range msgs {
		response[i] = PendingMessageResponse{
			ID:      msg.ID,
			Content: msg.Content,
			Status:  msg.Status,
		}

		if msg.CreatedAt.Valid {
			response[i].CreatedAt = msg.CreatedAt.Time
		}
	}

	render.JSON(w, r, response)
}

func (app *App) handleApproveMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	// (1) Get pending message
	pendingMsg, err := app.queries.GetPendingMessage(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}
		log.Printf("Error fetching pending message: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Use transaction for atomicity
	tx, err := app.db.Begin()
	if err != nil {
		log.Printf("Error starting transaction: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()

	qtx := app.queries.WithTx(tx)

	// (2) Copy message into 'messages' table
	_, err = qtx.CreateMessage(ctx, pendingMsg.Content)
	if err != nil {
		log.Printf("Error creating message: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// (3) Mark pending message as 'approved'
	err = qtx.UpdatePendingMessageStatus(ctx, db.UpdatePendingMessageStatusParams{
		ID:     pendingMsg.ID,
		Status: "approved",
	})
	if err != nil {
		log.Printf("Error updating message: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Printf("Error committing transaction: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (app *App) handleRejectMessage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid message ID", http.StatusBadRequest)
		return
	}

	// (1) Get pending message
	pendingMsg, err := app.queries.GetPendingMessage(ctx, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Message not found", http.StatusNotFound)
			return
		}
		log.Printf("Error fetching pending message: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	// (2) Mark pending message as 'rejected'
	err = app.queries.UpdatePendingMessageStatus(ctx, db.UpdatePendingMessageStatusParams{
		ID:     pendingMsg.ID,
		Status: "rejected",
	})
	if err != nil {
		log.Printf("Error updating message: %v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
