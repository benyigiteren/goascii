package main

import (
	"ascii/pkg/admin"
	"ascii/pkg/ascii"
	"ascii/pkg/db"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Embed templates and static files from the web folder
//go:embed web/static/* web/templates/*
var embedFS embed.FS

func main() {
	// 1. Initialize DB
	dbDir := "./data"
	dbPath := dbDir + "/db.json"
	database, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// 2. Set up sub-filesystem for clean embedding
	webFS, err := fs.Sub(embedFS, "web")
	if err != nil {
		log.Fatalf("Failed to create web sub-filesystem: %v", err)
	}

	// 3. Parse templates
	tmpl, err := template.ParseFS(webFS, "templates/*.html")
	if err != nil {
		log.Fatalf("Failed to parse embedded templates: %v", err)
	}

	// 4. Create admin server instance
	adminServer := &admin.AdminServer{
		DB:        database,
		Templates: tmpl,
		DbDir:     dbDir,
	}

	// 5. Initialize router (using Go 1.22+ ServeMux)
	mux := http.NewServeMux()

	// Static files serving
	mux.Handle("GET /static/", http.FileServer(http.FS(webFS)))

	// Admin Panel Routes
	mux.HandleFunc("GET /admin/register", adminServer.HandleRegister)
	mux.HandleFunc("POST /admin/register", adminServer.HandleRegister)
	mux.HandleFunc("GET /admin/login", adminServer.HandleLogin)
	mux.HandleFunc("POST /admin/login", adminServer.HandleLogin)
	mux.HandleFunc("GET /admin/logout", adminServer.HandleLogout)
	mux.HandleFunc("GET /admin/dashboard", adminServer.AuthMiddleware(adminServer.HandleDashboard))

	// JSON info endpoint (superadmin + animasyonlar)
	mux.HandleFunc("GET /api/admin/info", adminServer.AuthMiddleware(adminServer.HandleAdminInfo))
	mux.HandleFunc("GET /api/dashboard", adminServer.AuthMiddleware(adminServer.HandleAdminInfo))

	// Animation actions
	mux.HandleFunc("POST /admin/animations", adminServer.AuthMiddleware(adminServer.HandleCreateAnimation))
	mux.HandleFunc("POST /admin/animations/delete", adminServer.AuthMiddleware(adminServer.HandleDeleteAnimation))
	
	// Preview & API
	mux.HandleFunc("GET /admin/preview/{name}", adminServer.AuthMiddleware(adminServer.HandlePreview))
	mux.HandleFunc("GET /api/animation/{name}", adminServer.HandleAPIAnimationGet)

	// Help Endpoint (/help)
	mux.HandleFunc("GET /help", makeHelpHandler(database))

	// ASCII Streamer / Client Handler
	mux.HandleFunc("GET /{name}", makeStreamHandler(database, dbDir))
	
	// Root endpoint
	mux.HandleFunc("GET /", makeRootHandler(database))

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on https://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// detectClient checks if the request comes from curl, wget, or a web browser
func detectClient(r *http.Request) (string, bool) {
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	if strings.Contains(ua, "curl") {
		return "curl", true
	}
	if strings.Contains(ua, "wget") {
		return "wget", true
	}
	if strings.Contains(ua, "libwww") || strings.Contains(ua, "httpie") {
		return "curl", true
	}
	return "browser", false
}

// getClientIP retrieves client IP behind proxies or direct
func getClientIP(r *http.Request) string {
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

// makeHelpHandler returns JSON detailing available animations
func makeHelpHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		scheme := schemeFor(r)
		anims := database.GetAnimations()
		animList := make([]map[string]string, 0, len(anims))
		for _, a := range anims {
			animList = append(animList, map[string]string{
				"slug": a.Slug,
				"name": a.Name,
				"url":  scheme + "://" + r.Host + "/" + a.Slug,
			})
		}

		helpData := map[string]interface{}{
			"servis":       "go-ascii",
			"aciklama":     "Terminal uzerinden ASCII animasyon akis servisi.",
			"kullanim":     "curl -s " + scheme + "://" + r.Host + "/<slug>",
			"animasyonlar": animList,
			"parametreler": map[string]string{
				"w":     "Karakter genisligi (Ornek: ?w=100)",
				"h":     "Karakter yuksekligi (Ornek: ?h=30)",
				"color": "Renkleri kapatir (Ornek: ?color=false, varsayilan: true)",
			},
		}

		_ = json.NewEncoder(w).Encode(helpData)
	}
}

func schemeFor(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "https"
}

// makeRootHandler handles requests to "/"
func makeRootHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		clientType, isTerminal := detectClient(r)
		if !isTerminal {
			// Redirect browser users to the dashboard panel
			http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
			return
		}

		// Terminal clients (curl, wget, httpie) get the help JSON directly.
		startTime := time.Now()
		w.Header().Set("Content-Type", "application/json; charset=utf-8")

		scheme := schemeFor(r)
		anims := database.GetAnimations()
		animList := make([]map[string]string, 0, len(anims))
		for _, a := range anims {
			animList = append(animList, map[string]string{
				"slug": a.Slug,
				"name": a.Name,
				"url":  scheme + "://" + r.Host + "/" + a.Slug,
			})
		}

		helpData := map[string]interface{}{
			"servis":       "go-ascii",
			"aciklama":     "Terminal uzerinden ASCII animasyon akis servisi.",
			"kullanim":     "curl -s " + scheme + "://" + r.Host + "/<slug>",
			"animasyonlar": animList,
			"parametreler": map[string]string{
				"w":     "Karakter genisligi (Ornek: ?w=100)",
				"h":     "Karakter yuksekligi (Ornek: ?h=30)",
				"color": "Renkleri kapatir (Ornek: ?color=false, varsayilan: true)",
			},
		}

		_ = json.NewEncoder(w).Encode(helpData)

		// Add run log for root query
		_ = database.AddLog("root", getClientIP(r), r.Header.Get("User-Agent"), clientType, time.Since(startTime).Seconds())
	}
}

// makeStreamHandler handles GET /{name} for chunked ASCII terminal streaming
func makeStreamHandler(database *db.DB, dbDir string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := r.PathValue("name")
		
		// Verify animation exists
		animMeta, ok := database.GetAnimation(name)
		if !ok {
			// Unregistered route fallback
			clientType, isTerminal := detectClient(r)
			if isTerminal {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Bilinmeyen adres. Mevcut adresleri gormek icin /help adresini ziyaret edin.\n"))
				
				// Log the unknown query
				_ = database.AddLog("invalid:"+name, getClientIP(r), r.Header.Get("User-Agent"), clientType, 0)
			} else {
				// Redirect browser to help endpoint
				http.Redirect(w, r, "/help", http.StatusSeeOther)
			}
			return
		}

		// Browser redirect to preview player
		clientType, isTerminal := detectClient(r)
		if !isTerminal {
			http.Redirect(w, r, "/admin/preview/"+name, http.StatusSeeOther)
			return
		}

		// Increment database counter
		_ = database.IncrementAnimationRunCount(name)

		// Setup chunked streaming
		flusher, ok := w.(http.ResponseWriter).(http.Flusher)
		if !ok {
			flusher, ok = w.(http.Flusher)
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Transfer-Encoding", "chunked")
		w.Header().Set("Connection", "keep-alive")
		
		if ok {
			flusher.Flush()
		}

		// Read scaling dimensions
		width := 80
		height := 24
		if wQuery := r.URL.Query().Get("w"); wQuery != "" {
			if val, err := strconv.Atoi(wQuery); err == nil && val > 0 {
				width = val
			}
		}
		if hQuery := r.URL.Query().Get("h"); hQuery != "" {
			if val, err := strconv.Atoi(hQuery); err == nil && val > 0 {
				height = val
			}
		}

		// Disable ANSI colors if requested
		useColor := r.URL.Query().Get("color") != "false"

		// Set frame loops
		var customAnim *ascii.ConvertedAnimation
		var matrixState *ascii.MatrixState
		var fireState *ascii.FireState
		var hacker3State *ascii.Hacker3State

		if animMeta.Type == "custom" {
			var err error
			customAnim, err = ascii.LoadAnimationFromFile(dbDir, name)
			if err != nil {
				_, _ = fmt.Fprintf(w, "\n[Hata: %v]\n", err)
				return
			}
		} else if name == "matrix" {
			matrixState = ascii.NewMatrixState(width, height)
		} else if name == "hacker3" {
			hacker3State = ascii.NewHacker3State(width, height)
		}

		startTime := time.Now()
		ua := r.Header.Get("User-Agent")
		tick := 0

		// Streaming Loop
		for {
			select {
			case <-r.Context().Done():
				// Connection closed (Ctrl+C)
				duration := time.Since(startTime).Seconds()
				_ = database.AddLog(name, getClientIP(r), ua, clientType, duration)
				return
			default:
				// Clear screen & home cursor
				_, _ = w.Write([]byte("\033[2J\033[H"))

				var frameContent string
				delayMs := animMeta.FrameDelayMs

			switch animMeta.Type {
			case "procedural":
				switch name {
				case "earth":
					frameContent = ascii.GetEarthFrame(tick, width, height, useColor)
				case "matrix":
					frameContent, matrixState = ascii.GetMatrixFrame(tick, width, height, matrixState, useColor)
				case "donut":
					frameContent = ascii.GetDonutFrame(tick, width, height)
				case "cube":
					frameContent = ascii.GetCubeFrame(tick, width, height)
				case "fire":
					frameContent, fireState = ascii.GetFireFrame(tick, width, height, fireState, useColor)
				case "nyancat":
					frameContent = ascii.GetNyanCatFrame(tick, width, height, useColor)
				case "crewmate":
					frameContent = ascii.GetCrewmateFrame(tick, width, height, useColor)
				case "hacker1":
					frameContent = ascii.GetHacker1Frame(tick, width, height, useColor)
				case "hacker2":
					frameContent = ascii.GetHacker2Frame(tick, width, height, useColor)
				case "hacker3":
					frameContent, hacker3State = ascii.GetHacker3Frame(tick, width, height, hacker3State, useColor)
				case "doge":
					frameContent = ascii.GetDogeFrame(tick, width, height, useColor)
				case "dancer":
					frameContent = ascii.GetDancerFrame(tick, width, height, useColor)
				case "dab":
					frameContent = ascii.GetDabFrame(tick, width, height, useColor)
				case "rock":
					frameContent = ascii.GetRockFrame(tick, width, height, useColor)
				case "troll":
					frameContent = ascii.GetTrollFrame(tick, width, height, useColor)
				}
				case "custom":
					if customAnim != nil && len(customAnim.Frames) > 0 {
						frameIdx := tick % len(customAnim.Frames)
						frameContent = customAnim.Frames[frameIdx]
						if len(customAnim.Delays) > frameIdx {
							delayMs = customAnim.Delays[frameIdx]
						}
					}
				}

				// Write frame to stream
				_, _ = w.Write([]byte(frameContent))

				// Force flush
				if flusher != nil {
					flusher.Flush()
				}

				// Sleep for frame interval
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
				tick++
			}
		}
	}
}
