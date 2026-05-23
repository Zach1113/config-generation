package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brian/config-generation/backend/db"
	"github.com/brian/config-generation/backend/handlers"
	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		log.Fatal("JWT_SECRET environment variable is required")
	}

	database, err := db.Open(dsn)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer database.Close()

	// Register a gauge that reports the current number of open DB connections.
	// GaugeFunc samples the value on each Prometheus scrape — no manual updates needed.
	prometheus.MustRegister(prometheus.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "config_gen_db_open_connections",
			Help: "Number of open database connections.",
		},
		func() float64 { return float64(database.Stats().OpenConnections) },
	))

	if err := seedAdmin(database); err != nil {
		log.Fatalf("failed to seed admin user: %v", err)
	}

	router := handlers.NewRouterWithAuthConfig(database, handlers.AuthConfigFromEnv([]byte(jwtSecret)))

	addr := ":8080"
	if port := os.Getenv("PORT"); port != "" {
		addr = ":" + port
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// Start serving in a goroutine so the main goroutine can block on the
	// signal channel below.
	go func() {
		log.Printf("server listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Graceful shutdown: wait for SIGTERM (Kubernetes) or SIGINT (Ctrl-C),
	// then give in-flight requests up to 30 s to complete.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down server…")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("server forced to shutdown: %v", err)
	}
	log.Println("server stopped")
}

// seedAdmin creates the initial admin user if ADMIN_USERNAME / ADMIN_PASSWORD
// are set.  The INSERT uses ON CONFLICT DO NOTHING so that concurrent pod
// starts (minReplicas ≥ 2) do not race on the UNIQUE username constraint and
// crash with a UNIQUE violation.
func seedAdmin(database *sql.DB) error {
	username := os.Getenv("ADMIN_USERNAME")
	password := os.Getenv("ADMIN_PASSWORD")
	if username == "" || password == "" {
		log.Println("ADMIN_USERNAME/ADMIN_PASSWORD not set, skipping admin seed")
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	result, err := database.Exec(
		`INSERT INTO users (username, display_name, password_hash, superuser)
		 VALUES ($1, $2, $3, true)
		 ON CONFLICT (username) DO NOTHING`,
		username, "Administrator", string(hash),
	)
	if err != nil {
		return err
	}

	if n, _ := result.RowsAffected(); n > 0 {
		log.Printf("admin user %q created", username)
	}
	return nil
}
