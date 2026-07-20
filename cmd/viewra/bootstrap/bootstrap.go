package bootstrap

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	_ "net/http/pprof" // Register pprof handlers
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"github.com/mantonx/viewra/internal/app"
	appconfig "github.com/mantonx/viewra/internal/app/config"
	"github.com/mantonx/viewra/internal/pkg/logger"
	"github.com/mantonx/viewra/internal/version"
	"github.com/mantonx/viewra/web"
)

// Application holds all application dependencies and manages lifecycle.
// Config, Logger, Database handle runtime concerns.
// Container holds the dependency injection graph for business logic.
type Application struct {
	Config    *appconfig.Config
	Logger    *slog.Logger
	Database  *DatabaseConnection
	Container *app.Container
}

// Initialize sets up the application with all dependencies
func Initialize() (*Application, error) {
	// Load configuration from environment
	cfg, err := appconfig.Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize logger
	lgr := logger.New(cfg.Environment)
	lgr.Info("Starting ViewRA Media Server", "version", "0.0.1", "environment", cfg.Environment)

	// Initialize database
	dbConn, err := InitializeDatabaseFromConfig(&cfg.Database, lgr)
	if err != nil {
		return nil, err
	}

	// Run pre-container startup tasks (system profiling, migrations)
	ctx := context.Background()
	if err := RunPreContainerTasks(ctx, dbConn.DB, cfg, lgr); err != nil {
		dbConn.Close(lgr)
		return nil, err
	}

	// Initialize application container with all dependencies
	container := app.NewContainer(dbConn.DB, dbConn.Driver, cfg, lgr)

	// Run post-container startup tasks (reuses container's services)
	RunPostContainerTasks(ctx, container.UseCases.Library.Scan, lgr)

	// Add Swagger documentation endpoint
	container.Server.Router().GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Serve /web/manifest.json for Jellyfin WebOS app compatibility
	// The jellyfin-webos app hardcodes /web/manifest.json to discover the web client URL.
	container.Server.Router().GET("/web/manifest.json", func(c *gin.Context) {
		c.Data(http.StatusOK, "application/json", []byte(`{
			"shortname": "ViewRA",
			"name": "ViewRA Media Server",
			"start_url": "/webos/"
		}`))
	})

	// Serve embedded frontends in production (if available)
	if web.IsEmbedded() {
		// --- Web frontend (served at /) ---
		webFS, err := web.WebFS()
		if err != nil {
			lgr.Warn("Web frontend embedded but failed to load", "error", err)
		} else {
			lgr.Info("Serving web frontend at http://localhost:8080/")
			webHTTPFS := http.FS(webFS)

			indexHTML, err := webFS.Open("index.html")
			if err != nil {
				lgr.Error("Failed to read web index.html", "error", err)
			}
			webIndexContent, _ := io.ReadAll(indexHTML)
			indexHTML.Close()

			// --- TV frontend (served at /webos/) ---
			tvFS, tvErr := web.TvFS()
			var tvHTTPFS http.FileSystem
			var tvIndexContent []byte
			if tvErr != nil {
				lgr.Warn("TV frontend embedded but failed to load", "error", tvErr)
			} else {
				tvHTTPFS = http.FS(tvFS)
				tvIdx, openErr := tvFS.Open("index.html")
				if openErr != nil {
					lgr.Error("Failed to read tv/index.html", "error", openErr)
				} else {
					tvIndexContent, _ = io.ReadAll(tvIdx)
					tvIdx.Close()
				}
			}

			// Register TV frontend route BEFORE NoRoute
			if tvHTTPFS != nil {
				router := container.Server.Router()
				router.GET("/webos/*any", func(c *gin.Context) {
					path := c.Param("any")
					if strings.Contains(path, ".") {
						c.FileFromFS(path, tvHTTPFS)
						return
					}
					c.Data(http.StatusOK, "text/html; charset=utf-8", tvIndexContent)
				})
			}

			// Web frontend SPA fallback (catches all other paths)
			container.Server.Router().NoRoute(func(c *gin.Context) {
				path := c.Request.URL.Path
				// Skip ViewRA API paths (default 404)
				if isAPIPath(path) {
					return
				}
				// Log unhandled Jellyfin-style API endpoints
				if isJellyfinPath(path) {
					lgr.Warn("jellyfin: unhandled endpoint",
						"method", c.Request.Method,
						"path", c.Request.URL.String(),
					)
					c.JSON(404, gin.H{"error": "Not implemented"})
					return
				}
				// Redirect /web/index.html → /web/ so SPA loads at correct base route
				if strings.HasSuffix(path, "index.html") {
					c.Redirect(http.StatusMovedPermanently, "./")
					return
				}
				// Try to serve the file directly (for assets like .js, .css, images)
				if strings.Contains(path, ".") {
					fsPath := strings.TrimPrefix(path, "/web")
					c.FileFromFS(fsPath, webHTTPFS)
					return
				}
				// Log unmatched paths for debugging (catches e.g. /web/manifest.json before fix)
				lgr.Debug("frontend: serving SPA for unmatched path",
					"method", c.Request.Method,
					"path", c.Request.URL.String(),
				)
				// For SPA routes (no extension), serve index.html content directly
				c.Data(http.StatusOK, "text/html; charset=utf-8", webIndexContent)
			})
		}
	} else {
		lgr.Info("Frontend not embedded - development mode (use Vite on :5173)")
		container.Server.Router().NoRoute(func(c *gin.Context) {
			if isJellyfinPath(c.Request.URL.Path) {
				lgr.Warn("jellyfin: unhandled endpoint",
					"method", c.Request.Method,
					"path", c.Request.URL.String(),
				)
				c.JSON(404, gin.H{"error": "Not implemented"})
			}
		})
	}

	return &Application{
		Config:    cfg,
		Logger:    lgr,
		Database:  dbConn,
		Container: container,
	}, nil
}

// Run starts the HTTP server and handles graceful shutdown
func (a *Application) Run() error {
	// Ensure database is closed on exit
	defer a.Database.Close(a.Logger)

	// Start pprof server only in dev mode (security: exposes profiling data without auth)
	if os.Getenv("VIEWRA_DEV_MODE") == "1" {
		go func() {
			// Add endpoint to force memory release to OS
			http.HandleFunc("/debug/freeOSMemory", func(w http.ResponseWriter, r *http.Request) {
				debug.FreeOSMemory()
				w.Write([]byte("Memory released to OS\n"))
			})
			a.Logger.Info("pprof server starting (dev mode only)", "url", "http://localhost:6060/debug/pprof/")
			if err := http.ListenAndServe(":6060", nil); err != nil {
				a.Logger.Error("pprof server error", "error", err)
			}
		}()
	}

	// Start transcode queue if available
	if a.Container.TranscodeQueue != nil {
		ctx := context.Background()
		if err := a.Container.TranscodeQueue.Start(ctx); err != nil {
			a.Logger.Error("Failed to start transcode queue", "error", err)
		} else {
			a.Logger.Info("Transcode queue started")
		}
	}

	// Start scheduler service if available
	// (includes transcode cleanup, image cleanup, session cleanup tasks)
	if a.Container.SchedulerService != nil {
		ctx := context.Background()
		go func() {
			if err := a.Container.SchedulerService.Start(ctx); err != nil {
				a.Logger.Error("Scheduler error", "error", err)
			}
		}()
	}

	// Start server in goroutine
	go func() {
		a.Logger.Info("HTTP server starting",
			"port", a.Config.Server.Port,
			"swagger", fmt.Sprintf("http://localhost:%d/swagger/index.html", a.Config.Server.Port))
		if err := a.Container.Server.Start(); err != nil {
			a.Logger.Error("Server error", "error", err)
		}
	}()

	// Start background services AFTER HTTP server is listening
	// This ensures the UI is accessible even while heavy tasks run
	go a.Container.StartBackgroundServices(context.Background())

	// Give server a moment to bind, then log ready status
	time.Sleep(100 * time.Millisecond)
	a.logStartupReady()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit

	a.Logger.Info("Shutdown signal received", "signal", sig.String())

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := a.Container.Shutdown(ctx); err != nil {
		a.Logger.Error("Container forced shutdown", "error", err)
		return err
	}

	a.Logger.Info("Application stopped gracefully")
	return nil
}

// isAPIPath checks if a path is a ViewRA API endpoint
func isAPIPath(path string) bool {
	return strings.HasPrefix(path, "/api/") ||
		strings.HasPrefix(path, "/swagger/") ||
		strings.HasPrefix(path, "/health")
}

// isJellyfinPath checks if a path looks like a Jellyfin-compatible API endpoint.
// These are paths the Jellyfin handler registers routes for, but if they don't
// match, we log them so we can identify missed endpoints.
func isJellyfinPath(path string) bool {
	return strings.HasPrefix(path, "/Users/") ||
		strings.HasPrefix(path, "/System/") ||
		strings.HasPrefix(path, "/Items/") ||
		strings.HasPrefix(path, "/Videos/") ||
		strings.HasPrefix(path, "/Audio/") ||
		strings.HasPrefix(path, "/Shows/") ||
		strings.HasPrefix(path, "/Sessions/") ||
		strings.HasPrefix(path, "/Branding/") ||
		strings.HasPrefix(path, "/QuickConnect/") ||
		strings.HasPrefix(path, "/DisplayPreferences/") ||
		strings.HasPrefix(path, "/Search/") ||
		strings.HasPrefix(path, "/Collections/") ||
		strings.HasPrefix(path, "/LiveTv/") ||
		strings.HasPrefix(path, "/Notifications/") ||
		strings.HasPrefix(path, "/Packages/") ||
		strings.HasPrefix(path, "/Plugins/") ||
		strings.HasPrefix(path, "/HomeScreen/") ||
		strings.HasPrefix(path, "/MediaSegments/") ||
		strings.HasPrefix(path, "/Genres/") ||
		strings.HasPrefix(path, "/Persons/")
}

// logStartupReady prints a startup banner confirming all services are ready
func (a *Application) logStartupReady() {
	port := a.Config.Server.Port
	feStatus := "embedded"
	feURL := fmt.Sprintf("http://localhost:%d", port)
	if !web.IsEmbedded() {
		feStatus = "external (Vite :5173)"
		feURL = "http://localhost:5173"
	}

	// Dev credentials hint (only shown in development mode)
	devHint := ""
	if a.Config.Environment == "development" {
		devHint = "\n  Credentials: dev / devdev00 (auto-created)"
	}

	fmt.Fprintf(os.Stderr, `
========================================
  ViewRA ready
  Version:  %s
  Frontend: %s
  Database: %s%s
----------------------------------------
  App:     %s
  API:     http://localhost:%d/api
  Swagger: http://localhost:%d/swagger/index.html
  Health:  http://localhost:%d/health
========================================
`,
		version.Info(),
		feStatus,
		a.Config.Database.Driver,
		devHint,
		feURL,
		port, port, port,
	)
}
