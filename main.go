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

	// Help Endpoint (/help and /api/help)
	mux.HandleFunc("GET /help", makeHelpHandler(database))
	mux.HandleFunc("GET /api/help", makeHelpHandler(database))

	// ASCII Streamer / Client Handler
	mux.HandleFunc("GET /{name}", makeStreamHandler(database, dbDir))
	
	// Root endpoint
	mux.HandleFunc("GET /", makeRootHandler(database))

	// Primary listen address (default 8080, override with PORT or PORT_HTTP)
	port := os.Getenv("PORT")
	if port == "" {
		port = os.Getenv("PORT_HTTP")
	}
	if port == "" {
		port = "8080"
	}

	// Primary listen address (HTTP)
	primary := ":" + port
	log.Printf("Server starting on http://localhost%s", primary)

	// Optional secondary HTTP listener (e.g. expose on :80 in addition to
	// the container's :8080). Both HTTP listeners do NOT redirect to HTTPS;
	// the operator decides whether to run a TLS terminator in front.
	if extra := os.Getenv("HTTP_PORT_2"); extra != "" {
		go func() {
			addr := ":" + extra
			log.Printf("Also listening on %s", addr)
			if err := http.ListenAndServe(addr, mux); err != nil {
				log.Printf("Secondary HTTP listener on %s failed: %v", addr, err)
			}
		}()
	}

	// Optional HTTPS listener. Enable by setting:
	//   TLS_CERT_FILE=/path/to/fullchain.pem
	//   TLS_KEY_FILE=/path/to/privkey.pem
	//   HTTPS_PORT=443
	// When enabled, BOTH http and https keep working — we never 30x redirect
	// a client from http to https, so plain curl http://... always succeeds.
	certFile := os.Getenv("TLS_CERT_FILE")
	keyFile := os.Getenv("TLS_KEY_FILE")
	httpsPort := os.Getenv("HTTPS_PORT")
	if certFile != "" && keyFile != "" {
		if httpsPort == "" {
			httpsPort = "443"
		}
		go func() {
			addr := ":" + httpsPort
			log.Printf("TLS enabled, also listening on https://localhost%s", addr)
			if err := http.ListenAndServeTLS(addr, certFile, keyFile, mux); err != nil {
				log.Printf("TLS listener on %s failed: %v", addr, err)
			}
		}()
	}

	if err := http.ListenAndServe(primary, mux); err != nil {
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

		_, isTerminal := detectClient(r)
		if isTerminal || strings.Contains(r.Header.Get("Accept"), "application/json") {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			enc := json.NewEncoder(w)
			enc.SetIndent("", "  ")
			_ = enc.Encode(helpData)
			return
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write([]byte(renderLandingHTML(helpData, scheme, r.Host)))
	}
}

func schemeFor(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	if r.Header.Get("X-Forwarded-Ssl") == "on" {
		return "https"
	}
	// Default to http so plain http://ascii.yigiteren.org works
	// without being redirected to https.
	return "http"
}

func renderLandingHTML(data map[string]interface{}, scheme, host string) string {
	var sb strings.Builder
	sb.WriteString(`<!DOCTYPE html>
<html lang="tr">
<head>
<meta charset="UTF-8">
<title>go-ascii &mdash; ASCII Animasyon Servisi</title>
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<style>
body{background:#0b0b0b;color:#d4d4d4;font-family:'JetBrains Mono',ui-monospace,monospace;margin:0;padding:32px;line-height:1.5}
h1{color:#9ed68a;margin:0 0 8px 0;font-size:1.6rem}
.sub{color:#888;margin-bottom:24px}
a{color:#9ed68a;text-decoration:none}
a:hover{text-decoration:underline}
pre{background:#161616;border:1px solid #262626;padding:16px;border-radius:6px;overflow:auto;font-size:0.85rem;color:#b6f0c4}
table{width:100%;border-collapse:collapse;margin-top:16px;font-size:0.85rem}
th,td{text-align:left;padding:8px 10px;border-bottom:1px solid #222}
th{color:#888;font-weight:600}
tr:hover{background:#141414}
code{background:#1a1a1a;padding:2px 6px;border-radius:3px;color:#ffd28b}
.btn{display:inline-block;padding:6px 12px;background:#1a1a1a;border:1px solid #333;border-radius:4px;color:#9ed68a;font-size:0.8rem}
.btn:hover{background:#222}
</style>
</head>
<body>
<h1>go-ascii</h1>
<div class="sub">`)
	sb.WriteString(fmt.Sprint(data["aciklama"]))
	sb.WriteString(`</div>

<h2>Nasil kullanilir</h2>
<pre>curl -s `)
	sb.WriteString(scheme)
	sb.WriteString("://")
	sb.WriteString(host)
	sb.WriteString(`/&lt;slug&gt;</pre>

<h2>Mevcut animasyonlar</h2>
<table>
<thead><tr><th>Yayin</th><th>Slug</th><th>URL</th></tr></thead>
<tbody>`)

	if list, ok := data["animasyonlar"].([]map[string]string); ok {
		for _, a := range list {
			sb.WriteString("<tr><td>")
			sb.WriteString(a["name"])
			sb.WriteString("</td><td><code>/")
			sb.WriteString(a["slug"])
			sb.WriteString("</code></td><td><a href=\"")
			sb.WriteString(a["url"])
			sb.WriteString("\">")
			sb.WriteString(a["url"])
			sb.WriteString("</a></td></tr>")
		}
	}

	sb.WriteString(`</tbody></table>

<h2>Sorgu parametreleri</h2>
<ul>
<li><code>?w=100</code> &mdash; Karakter genisligi (varsayilan 80)</li>
<li><code>?h=30</code> &mdash; Karakter yuksekligi (varsayilan 24)</li>
<li><code>?color=false</code> &mdash; Renkleri kapatir</li>
</ul>

<p style="margin-top:32px;"><a class="btn" href="/api/help">JSON yardim (Accept: application/json)</a> &nbsp; <a class="btn" href="/admin/dashboard">Yonetim Paneli</a></p>

</body>
</html>`)

	return sb.String()
}

// makeRootHandler handles requests to "/"
func makeRootHandler(database *db.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		startTime := time.Now()
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

		// Browser istekleri icin JSON + minimal HTML landing
		clientType, isTerminal := detectClient(r)
		if !isTerminal {
			acceptsJSON := strings.Contains(r.Header.Get("Accept"), "application/json")
			if acceptsJSON {
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				enc := json.NewEncoder(w)
				enc.SetIndent("", "  ")
				_ = enc.Encode(helpData)
			} else {
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				_, _ = w.Write([]byte(renderLandingHTML(helpData, scheme, r.Host)))
			}
			_ = database.AddLog("root", getClientIP(r), r.Header.Get("User-Agent"), clientType, time.Since(startTime).Seconds())
			return
		}

		// Terminal clients (curl, wget, httpie) get the help JSON directly.
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		_ = enc.Encode(helpData)

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
			clientType, isTerminal := detectClient(r)
			if isTerminal || strings.Contains(r.Header.Get("Accept"), "application/json") {
				w.Header().Set("Content-Type", "text/plain; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte("Bilinmeyen adres. Mevcut adresleri gormek icin /help adresini ziyaret edin.\n"))
				_ = database.AddLog("invalid:"+name, getClientIP(r), r.Header.Get("User-Agent"), clientType, 0)
			} else {
				// Browser: redirect yerine inline HTML yardim sayfasi goster
				w.Header().Set("Content-Type", "text/html; charset=utf-8")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`<!DOCTYPE html><html lang="tr"><head><meta charset="UTF-8"><title>Yayin bulunamadi</title>
<style>body{background:#0b0b0b;color:#d4d4d4;font-family:monospace;padding:32px;line-height:1.5}a{color:#9ed68a}</style>
</head><body><h1 style="color:#ff7b7b">404 - Yayin bulunamadi</h1>
<p>"/` + name + `" bilinen bir yayin degil. <a href="/help">Mevcut yayinlar icin /help</a> adresini ziyaret edin.</p>
</body></html>`))
				_ = database.AddLog("invalid:"+name, getClientIP(r), r.Header.Get("User-Agent"), clientType, 0)
			}
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
		clientType, _ := detectClient(r)
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
				case "kitty":
					frameContent = ascii.GetKittyFrame(tick, width, height, useColor)
				case "parrot":
					frameContent = ascii.GetParrotFrame(tick, width, height, useColor)
				case "coin":
					frameContent = ascii.GetCoinLiveFrame(tick, width, height, useColor)
				case "forrest":
					frameContent = ascii.GetForrestFrame(tick, width, height, useColor)
				case "bomb":
					frameContent = ascii.GetBombFrame(tick, width, height, useColor)
				case "nyan":
					frameContent = ascii.GetNyanLiveFrame(tick, width, height, useColor)
				case "purdue":
					frameContent = ascii.GetPurdueFrame(tick, width, height, useColor)
				case "india":
					frameContent = ascii.GetIndiaFrame(tick, width, height, useColor)
				case "knot":
					frameContent = ascii.GetKnotFrame(tick, width, height, useColor)
				case "maxwell":
					frameContent = ascii.GetMaxwellFrame(tick, width, height, useColor)
				case "astrand":
					frameContent = ascii.GetAstrendFrame(tick, width, height, useColor)
				case "brittany":
					frameContent = ascii.GetBrittanyFrame(tick, width, height, useColor)
				case "batman":
					frameContent = ascii.GetBatmanFrame(tick, width, height, useColor)
				case "batman-running":
					frameContent = ascii.GetBatmanRunningFrame(tick, width, height, useColor)
				case "bnr":
					frameContent = ascii.GetBNRFrame(tick, width, height, useColor)
				case "spidyswing":
					frameContent = ascii.GetSpidyswingFrame(tick, width, height, useColor)
				case "rick":
					frameContent = ascii.GetRickLiveFrame(tick, width, height, useColor)
				case "can-you-hear-me":
					frameContent = ascii.GetCanYouHearMeFrame(tick, width, height, useColor)
				case "earth-live":
					frameContent = ascii.GetEarthLiveFrame(tick, width, height, useColor)
				case "nyan-live":
					frameContent = ascii.GetNyanLiveFrame(tick, width, height, useColor)
				case "donut-live":
					frameContent = ascii.GetDonutLiveFrame(tick, width, height, useColor)
				case "hes":
					frameContent = ascii.GetHESFrame(tick, width, height, useColor)
				case "torus-knot":
					frameContent = ascii.GetKnotFrame(tick, width, height, useColor)
				case "as":
					frameContent = ascii.GetAstrendFrame(tick, width, height, useColor)
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
