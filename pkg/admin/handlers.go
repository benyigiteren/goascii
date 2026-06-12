package admin

import (
	"ascii/pkg/ascii"
	"ascii/pkg/db"
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type AdminServer struct {
	DB        *db.DB
	Templates *template.Template
	DbDir     string
}

// AuthMiddleware wraps handlers to require a valid session
func (s *AdminServer) AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sm := GetSessionManager()
		if _, ok := sm.GetUsername(r); !ok {
			http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}

// HandleRegister handles superadmin creation
func (s *AdminServer) HandleRegister(w http.ResponseWriter, r *http.Request) {
	usersCount := s.DB.GetUsersCount()
	if usersCount > 0 {
		http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		err := s.Templates.ExecuteTemplate(w, "register.html", map[string]interface{}{
			"Title": "Süperadmin Oluştur",
		})
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if r.Method == http.MethodPost {
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")
		confirm := r.FormValue("confirm_password")

		if username == "" || password == "" {
			s.renderRegisterError(w, "Kullanıcı adı ve şifre boş bırakılamaz.")
			return
		}

		if password != confirm {
			s.renderRegisterError(w, "Şifreler uyuşmuyor.")
			return
		}

		if len(password) < 6 {
			s.renderRegisterError(w, "Şifre en az 6 karakter olmalıdır.")
			return
		}

		hash, salt, err := HashPassword(password)
		if err != nil {
			s.renderRegisterError(w, "Şifre işlenirken hata oluştu.")
			return
		}

		if err := s.DB.CreateUser(username, hash, salt); err != nil {
			s.renderRegisterError(w, "Kullanıcı oluşturulurken hata: "+err.Error())
			return
		}

		http.Redirect(w, r, "/admin/login?registered=true", http.StatusSeeOther)
	}
}

func (s *AdminServer) renderRegisterError(w http.ResponseWriter, msg string) {
	_ = s.Templates.ExecuteTemplate(w, "register.html", map[string]interface{}{
		"Title": "Süperadmin Oluştur",
		"Error": msg,
	})
}

// HandleLogin handles admin authentication
func (s *AdminServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	usersCount := s.DB.GetUsersCount()
	if usersCount == 0 {
		http.Redirect(w, r, "/admin/register", http.StatusSeeOther)
		return
	}

	// Check if already logged in
	sm := GetSessionManager()
	if _, ok := sm.GetUsername(r); ok {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	if r.Method == http.MethodGet {
		registered := r.URL.Query().Get("registered") == "true"
		_ = s.Templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
			"Title":      "Giriş Yap",
			"Success":    registered,
			"Registered": registered,
		})
		return
	}

	if r.Method == http.MethodPost {
		username := strings.TrimSpace(r.FormValue("username"))
		password := r.FormValue("password")

		user, ok := s.DB.GetUser(username)
		if !ok || !VerifyPassword(password, user.PasswordHash, user.Salt) {
			_ = s.Templates.ExecuteTemplate(w, "login.html", map[string]interface{}{
				"Title": "Giriş Yap",
				"Error": "Geçersiz kullanıcı adı veya şifre.",
			})
			return
		}

		sm.CreateSession(username, w)
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
	}
}

// HandleLogout logs the user out
func (s *AdminServer) HandleLogout(w http.ResponseWriter, r *http.Request) {
	sm := GetSessionManager()
	sm.DestroySession(r, w)
	http.Redirect(w, r, "/admin/login", http.StatusSeeOther)
}

// HandleDashboard renders dashboard statistics and lists animations
func (s *AdminServer) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	username, _ := GetSessionManager().GetUsername(r)
	anims := s.DB.GetAnimations()
	recentLogs := s.DB.GetLogs(15)

	totalRuns := s.DB.GetTotalRunCount()
	uniqueIPs := s.DB.GetUniqueIPsCount()
	clientCounts := s.DB.GetClientTypeCounts()

	// Calculate average duration
	allLogs := s.DB.GetLogs(5000)
	var avgDuration float64
	if len(allLogs) > 0 {
		var sum float64
		for _, log := range allLogs {
			sum += log.DurationSeconds
		}
		avgDuration = sum / float64(len(allLogs))
	}

		// Build host address for display
		host := r.Host
		if !strings.Contains(host, "://") {
			proto := "https"
			if r.TLS != nil {
				proto = "https"
			} else if r.Header.Get("X-Forwarded-Proto") != "" {
				proto = r.Header.Get("X-Forwarded-Proto")
			} else if r.Header.Get("X-Forwarded-Ssl") == "on" {
				proto = "https"
			} else {
				proto = "http"
			}
			host = proto + "://" + host
		}

	// Dynamic errors/success decoding from query string
	errQuery := r.URL.Query().Get("error")
	successQuery := r.URL.Query().Get("success")

	data := map[string]interface{}{
		"Title":        "Kontrol Paneli",
		"Username":     username,
		"Animations":   anims,
		"Logs":         recentLogs,
		"TotalRuns":    totalRuns,
		"UniqueIPs":    uniqueIPs,
		"CurlCount":    clientCounts["curl"],
		"WgetCount":    clientCounts["wget"],
		"BrowserCount": clientCounts["browser"],
		"AvgDuration":  fmt.Sprintf("%.2fs", avgDuration),
		"Host":         host,
		"Error":        errQuery,
		"Success":      successQuery,
	}

	_ = s.Templates.ExecuteTemplate(w, "dashboard.html", data)
}

// HandleAdminInfo returns a JSON snapshot of the current admin user, role,
// and the full animation list. Useful for the first registered user to
// inspect what is available right after signing up.
func (s *AdminServer) HandleAdminInfo(w http.ResponseWriter, r *http.Request) {
	username, _ := GetSessionManager().GetUsername(r)

	role := "user"
	createdAt := time.Time{}
	if u, ok := s.DB.GetUser(username); ok {
		role = u.Role
		createdAt = u.CreatedAt
	}

	anims := s.DB.GetAnimations()
	animList := make([]map[string]interface{}, 0, len(anims))
	for _, a := range anims {
		animList = append(animList, map[string]interface{}{
			"slug":          a.Slug,
			"name":          a.Name,
			"type":          a.Type,
			"frame_delay_ms": a.FrameDelayMs,
			"frames_count":  a.FramesCount,
			"run_count":     a.RunCount,
			"created_by":    a.CreatedBy,
			"created_at":    a.CreatedAt,
			"url":           "https://" + r.Host + "/" + a.Slug,
		})
	}

	totalUsers := s.DB.GetUsersCount()
	totalRuns := s.DB.GetTotalRunCount()
	uniqueIPs := s.DB.GetUniqueIPsCount()
	clientCounts := s.DB.GetClientTypeCounts()

	proto := "http"
	if r.TLS != nil {
		proto = "https"
	} else if r.Header.Get("X-Forwarded-Proto") != "" {
		proto = r.Header.Get("X-Forwarded-Proto")
	} else if r.Header.Get("X-Forwarded-Ssl") == "on" {
		proto = "https"
	}

	payload := map[string]interface{}{
		"current_user": map[string]interface{}{
			"username":   username,
			"role":       role,
			"created_at": createdAt,
		},
		"server": map[string]interface{}{
			"host":      r.Host,
			"scheme":    proto,
			"base_url":  proto + "://" + r.Host,
		},
		"stats": map[string]interface{}{
			"total_users":   totalUsers,
			"total_runs":    totalRuns,
			"unique_ips":    uniqueIPs,
			"client_types":  clientCounts,
		},
		"animations": animList,
		"endpoints": map[string]string{
			"help":           proto + "://" + r.Host + "/help",
			"register":       proto + "://" + r.Host + "/admin/register",
			"login":          proto + "://" + r.Host + "/admin/login",
			"dashboard":      proto + "://" + r.Host + "/admin/dashboard",
			"admin_info":     proto + "://" + r.Host + "/api/admin/info",
			"api_dashboard":  proto + "://" + r.Host + "/api/dashboard",
		},
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(payload)
}

// HandleCreateAnimation uploads and converts GIF or MP4
func (s *AdminServer) HandleCreateAnimation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	username, _ := GetSessionManager().GetUsername(r)

	// Parse multi-part form (max 15MB to allow slightly larger MP4s)
	err := r.ParseMultipartForm(15 << 20)
	if err != nil {
		http.Redirect(w, r, "/admin/dashboard?error=Dosya+boyutu+cok+buyuk+(maksimum+15MB)", http.StatusSeeOther)
		return
	}

	name := strings.TrimSpace(r.FormValue("name"))
	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	widthStr := r.FormValue("width")
	cropCenter := r.FormValue("crop_center") == "on"

	// Validate fields
	if name == "" || slug == "" {
		http.Redirect(w, r, "/admin/dashboard?error=Isim+ve+Yol+Adi+(Slug)+zorunludur", http.StatusSeeOther)
		return
	}

	// Validate slug: alphanumeric and dashes only
	isValidSlug, _ := regexp.MatchString("^[a-z0-9-]+$", slug)
	if !isValidSlug {
		http.Redirect(w, r, "/admin/dashboard?error=Yol+adi+sadece+harf,+sayi+ve+tire+icerebilir", http.StatusSeeOther)
		return
	}

	// Validate builtin names
	if slug == "earth" || slug == "matrix" || slug == "donut" || slug == "admin" || slug == "api" || slug == "help" {
		http.Redirect(w, r, "/admin/dashboard?error=Yol+adi+sistem+tarafindan+rezerve+edilmistir", http.StatusSeeOther)
		return
	}

	width := 80
	if widthStr != "" {
		if wVal, err := strconv.Atoi(widthStr); err == nil && wVal > 0 {
			width = wVal
		}
	}
	if width > 160 {
		width = 160 // Cap width
	}

	file, header, err := r.FormFile("gif_file")
	if err != nil {
		http.Redirect(w, r, "/admin/dashboard?error=Lutfen+bir+GIF+veya+MP4+dosyasi+secin", http.StatusSeeOther)
		return
	}
	defer file.Close()

	// Read file contents
	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		http.Redirect(w, r, "/admin/dashboard?error=Yuklenen+dosya+okunamadi", http.StatusSeeOther)
		return
	}

	ext := strings.ToLower(filepath.Ext(header.Filename))

	var converted *ascii.ConvertedAnimation

	if ext == ".mp4" {
		converted, err = ascii.ConvertMP4(slug, name, buf.Bytes(), width, cropCenter)
	} else if ext == ".gif" {
		converted, err = ascii.ConvertGIF(slug, name, buf.Bytes(), width, cropCenter)
	} else {
		http.Redirect(w, r, "/admin/dashboard?error=Gecersiz+dosya+turu.+Lutfen+.gif+veya+.mp4+yukleyin.", http.StatusSeeOther)
		return
	}

	if err != nil {
		http.Redirect(w, r, "/admin/dashboard?error=Donusturme+hatasi:+"+err.Error(), http.StatusSeeOther)
		return
	}

	// Save converted animation JSON to disk
	if err := ascii.SaveAnimationToFile(s.DbDir, converted); err != nil {
		http.Redirect(w, r, "/admin/dashboard?error=Animasyon+dosyasi+kaydedilemedi", http.StatusSeeOther)
		return
	}

	// Save to DB metadata
	err = s.DB.CreateAnimation(slug, name, "custom", converted.FrameDelayMs, len(converted.Frames), username)
	if err != nil {
		// Clean up file if DB save failed
		animFile := filepath.Join(s.DbDir, "animations", slug+".json")
		_ = os.Remove(animFile)
		http.Redirect(w, r, "/admin/dashboard?error=Animasyon+kaydedilemedi:+"+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/dashboard?success=Animasyon+basariyla+olusturuldu", http.StatusSeeOther)
}

// HandleDeleteAnimation handles deleting custom animation
func (s *AdminServer) HandleDeleteAnimation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/admin/dashboard", http.StatusSeeOther)
		return
	}

	slug := r.FormValue("slug")
	if err := s.DB.DeleteAnimation(slug); err != nil {
		http.Redirect(w, r, "/admin/dashboard?error="+err.Error(), http.StatusSeeOther)
		return
	}

	http.Redirect(w, r, "/admin/dashboard?success=Animasyon+basariyla+silindi", http.StatusSeeOther)
}

// HandlePreview renders the browser-based terminal player
func (s *AdminServer) HandlePreview(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("name")
	anim, ok := s.DB.GetAnimation(slug)
	if !ok {
		http.Error(w, "Animasyon bulunamadı", http.StatusNotFound)
		return
	}

	data := map[string]interface{}{
		"Title":        "Önizleme: " + anim.Name,
		"Slug":         slug,
		"Name":         anim.Name,
		"FrameDelayMs": anim.FrameDelayMs,
	}

	_ = s.Templates.ExecuteTemplate(w, "player.html", data)
}

// HandleAPIAnimationGet returns the JSON frames of an animation for browser playback
func (s *AdminServer) HandleAPIAnimationGet(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("name")
	animMeta, ok := s.DB.GetAnimation(slug)
	if !ok {
		http.Error(w, "Animasyon bulunamadı", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	if animMeta.Type == "procedural" {
		var frames []string
		var delays []int

		count := 50
		width := 80
		height := 24

		var matrixState *ascii.MatrixState
		var fireState *ascii.FireState
		var hacker3State *ascii.Hacker3State

		for tick := 0; tick < count; tick++ {
			var frame string
			switch slug {
			case "earth":
				frame = ascii.GetEarthFrame(tick, width, height, false)
			case "matrix":
				frame, matrixState = ascii.GetMatrixFrame(tick, width, height, matrixState, false)
			case "donut":
				frame = ascii.GetDonutFrame(tick, width, height)
			case "cube":
				frame = ascii.GetCubeFrame(tick, width, height)
			case "fire":
				frame, fireState = ascii.GetFireFrame(tick, width, height, fireState, false)
			case "nyancat":
				frame = ascii.GetNyanCatFrame(tick, width, height, false)
			case "crewmate":
				frame = ascii.GetCrewmateFrame(tick, width, height, false)
			case "kitty":
				frame = ascii.GetKittyFrame(tick, width, height, false)
			case "parrot":
				frame = ascii.GetParrotFrame(tick, width, height, false)
			case "coin":
				frame = ascii.GetCoinLiveFrame(tick, width, height, false)
			case "forrest":
				frame = ascii.GetForrestFrame(tick, width, height, false)
			case "bomb":
				frame = ascii.GetBombFrame(tick, width, height, false)
			case "nyan":
				frame = ascii.GetNyanLiveFrame(tick, width, height, false)
			case "purdue":
				frame = ascii.GetPurdueFrame(tick, width, height, false)
			case "india":
				frame = ascii.GetIndiaFrame(tick, width, height, false)
			case "knot":
				frame = ascii.GetKnotFrame(tick, width, height, false)
			case "maxwell":
				frame = ascii.GetMaxwellFrame(tick, width, height, false)
			case "astrand":
				frame = ascii.GetAstrendFrame(tick, width, height, false)
			case "brittany":
				frame = ascii.GetBrittanyFrame(tick, width, height, false)
			case "batman":
				frame = ascii.GetBatmanFrame(tick, width, height, false)
			case "batman-running":
				frame = ascii.GetBatmanRunningFrame(tick, width, height, false)
			case "bnr":
				frame = ascii.GetBNRFrame(tick, width, height, false)
			case "spidyswing":
				frame = ascii.GetSpidyswingFrame(tick, width, height, false)
			case "rick":
				frame = ascii.GetRickLiveFrame(tick, width, height, false)
			case "can-you-hear-me":
				frame = ascii.GetCanYouHearMeFrame(tick, width, height, false)
			case "earth-live":
				frame = ascii.GetEarthLiveFrame(tick, width, height, false)
			case "nyan-live":
				frame = ascii.GetNyanLiveFrame(tick, width, height, false)
			case "donut-live":
				frame = ascii.GetDonutLiveFrame(tick, width, height, false)
			case "hes":
				frame = ascii.GetHESFrame(tick, width, height, false)
			case "torus-knot":
				frame = ascii.GetKnotFrame(tick, width, height, false)
			case "as":
				frame = ascii.GetAstrendFrame(tick, width, height, false)
			case "hacker1":
				frame = ascii.GetHacker1Frame(tick, width, height, false)
			case "hacker2":
				frame = ascii.GetHacker2Frame(tick, width, height, false)
			case "hacker3":
				frame, hacker3State = ascii.GetHacker3Frame(tick, width, height, hacker3State, false)
			case "doge":
				frame = ascii.GetDogeFrame(tick, width, height, false)
			case "dancer":
				frame = ascii.GetDancerFrame(tick, width, height, false)
			case "dab":
				frame = ascii.GetDabFrame(tick, width, height, false)
			case "rock":
				frame = ascii.GetRockFrame(tick, width, height, false)
			case "troll":
				frame = ascii.GetTrollFrame(tick, width, height, false)
			}
			frames = append(frames, frame)
			delays = append(delays, animMeta.FrameDelayMs)
		}

		resp := map[string]interface{}{
			"slug":           slug,
			"name":           animMeta.Name,
			"frame_delay_ms": animMeta.FrameDelayMs,
			"delays":         delays,
			"frames":         frames,
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}

	// Custom animations are stored as JSON files on disk
	anim, err := ascii.LoadAnimationFromFile(s.DbDir, slug)
	if err != nil {
		http.Error(w, "Animasyon verileri yüklenemedi: "+err.Error(), http.StatusInternalServerError)
		return
	}

	_ = json.NewEncoder(w).Encode(anim)
}
