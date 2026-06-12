package db

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type User struct {
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	Salt         string    `json:"salt"`
	Role         string    `json:"role"` // "superadmin" veya "user"
	CreatedAt    time.Time `json:"created_at"`
}

type Animation struct {
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Type         string    `json:"type"` // "procedural" or "custom"
	FrameDelayMs int       `json:"frame_delay_ms"`
	FramesCount  int       `json:"frames_count"`
	CreatedAt    time.Time `json:"created_at"`
	CreatedBy    string    `json:"created_by"`
	RunCount     int       `json:"run_count"`
}

type StatLog struct {
	Timestamp       time.Time `json:"timestamp"`
	AnimationSlug   string    `json:"animation_slug"`
	ClientIP        string    `json:"client_ip"`
	UserAgent       string    `json:"user_agent"`
	ClientType      string    `json:"client_type"` // "curl", "wget", "browser"
	DurationSeconds float64   `json:"duration_seconds"`
}

type DBState struct {
	Users      []User      `json:"users"`
	Animations []Animation `json:"animations"`
	Logs       []StatLog   `json:"logs"`
}

type DB struct {
	dbPath string
	mu     sync.RWMutex
	state  DBState
}

// InitDB initializes the database and returns a DB instance
func InitDB(dbPath string) (*DB, error) {
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, err
	}

	// Create animation data folder too
	animDir := filepath.Join(dbDir, "animations")
	if err := os.MkdirAll(animDir, 0755); err != nil {
		return nil, err
	}

	db := &DB{dbPath: dbPath}
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		// Create default database state with 10 built-in mathematical animations
		db.state = DBState{
			Users: []User{},
			Animations: []Animation{
				{
					Slug:         "earth",
					Name:         "3D Dönen Dünya Küresi",
					Type:         "procedural",
					FrameDelayMs: 60,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "matrix",
					Name:         "Matrix Dijital Yağmur",
					Type:         "procedural",
					FrameDelayMs: 50,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "donut",
					Name:         "3D Dönen Donut",
					Type:         "procedural",
					FrameDelayMs: 40,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "cube",
					Name:         "3D Dönen Tel Kafes Küp",
					Type:         "procedural",
					FrameDelayMs: 50,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "fire",
					Name:         "Prosedürel Doom Ateş Simülasyonu",
					Type:         "procedural",
					FrameDelayMs: 45,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "nyancat",
					Name:         "Nyan Cat - Pop-Tart Kedicik",
					Type:         "procedural",
					FrameDelayMs: 70,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "crewmate",
					Name:         "Among Us Crewmate - Yürüyen Karakter",
					Type:         "procedural",
					FrameDelayMs: 60,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "hacker1",
					Name:         "Hacker - Kayan Kaynak Kodu",
					Type:         "procedural",
					FrameDelayMs: 60,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "hacker2",
					Name:         "Hacker - Canlı Terminal Komutları",
					Type:         "procedural",
					FrameDelayMs: 80,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "hacker3",
					Name:         "Hacker - Binary Glitch Yağmuru",
					Type:         "procedural",
					FrameDelayMs: 50,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "doge",
					Name:         "Meme - Doge (wow such ascii)",
					Type:         "procedural",
					FrameDelayMs: 80,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "dancer",
					Name:         "Meme - Dansçı Stick Figure",
					Type:         "procedural",
					FrameDelayMs: 90,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "dab",
					Name:         "Meme - Dab Stick Figure",
					Type:         "procedural",
					FrameDelayMs: 100,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "rock",
					Name:         "Meme - Rock On El",
					Type:         "procedural",
					FrameDelayMs: 70,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
				{
					Slug:         "troll",
					Name:         "Meme - Trollface (PROBLEM?)",
					Type:         "procedural",
					FrameDelayMs: 80,
					FramesCount:  0,
					CreatedAt:    time.Now(),
					CreatedBy:    "system",
					RunCount:     0,
				},
			},
			Logs: []StatLog{},
		}
		if err := db.save(); err != nil {
			return nil, err
		}
	} else {
		// Read existing database
		data, err := os.ReadFile(dbPath)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(data, &db.state); err != nil {
			return nil, err
		}
		// Ensure default built-in mathematical animations exist
		db.ensureBuiltins()
	}

	return db, nil
}

func (db *DB) ensureBuiltins() {
	builtins := map[string]Animation{
		"earth": {
			Slug:         "earth",
			Name:         "3D Dönen Dünya Küresi",
			Type:         "procedural",
			FrameDelayMs: 60,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"matrix": {
			Slug:         "matrix",
			Name:         "Matrix Dijital Yağmur",
			Type:         "procedural",
			FrameDelayMs: 50,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"donut": {
			Slug:         "donut",
			Name:         "3D Dönen Donut",
			Type:         "procedural",
			FrameDelayMs: 40,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"cube": {
			Slug:         "cube",
			Name:         "3D Dönen Tel Kafes Küp",
			Type:         "procedural",
			FrameDelayMs: 50,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"life": {
			Slug:         "life",
			Name:         "Conway's Game of Life Simülasyonu",
			Type:         "procedural",
			FrameDelayMs: 120,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"fire": {
			Slug:         "fire",
			Name:         "Prosedürel Doom Ateş Simülasyonu",
			Type:         "procedural",
			FrameDelayMs: 45,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"nyancat": {
			Slug:         "nyancat",
			Name:         "Nyan Cat - Pop-Tart Kedicik",
			Type:         "procedural",
			FrameDelayMs: 70,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"crewmate": {
			Slug:         "crewmate",
			Name:         "Among Us Crewmate - Yürüyen Karakter",
			Type:         "procedural",
			FrameDelayMs: 60,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"hacker1": {
			Slug:         "hacker1",
			Name:         "Hacker - Kayan Kaynak Kodu",
			Type:         "procedural",
			FrameDelayMs: 60,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"hacker2": {
			Slug:         "hacker2",
			Name:         "Hacker - Canlı Terminal Komutları",
			Type:         "procedural",
			FrameDelayMs: 80,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"hacker3": {
			Slug:         "hacker3",
			Name:         "Hacker - Binary Glitch Yağmuru",
			Type:         "procedural",
			FrameDelayMs: 50,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"doge": {
			Slug:         "doge",
			Name:         "Meme - Doge (wow such ascii)",
			Type:         "procedural",
			FrameDelayMs: 80,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"dancer": {
			Slug:         "dancer",
			Name:         "Meme - Dansçı Stick Figure",
			Type:         "procedural",
			FrameDelayMs: 90,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"dab": {
			Slug:         "dab",
			Name:         "Meme - Dab Stick Figure",
			Type:         "procedural",
			FrameDelayMs: 100,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"rock": {
			Slug:         "rock",
			Name:         "Meme - Rock On El",
			Type:         "procedural",
			FrameDelayMs: 70,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
		"troll": {
			Slug:         "troll",
			Name:         "Meme - Trollface (PROBLEM?)",
			Type:         "procedural",
			FrameDelayMs: 80,
			FramesCount:  0,
			CreatedAt:    time.Now(),
			CreatedBy:    "system",
		},
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	modified := false

	// Drop procedural animations that are no longer in the built-in set
	// (e.g. life, plasma, heart, clock, fractal, rickroll). Custom uploads are kept.
	var cleanAnims []Animation
	for _, anim := range db.state.Animations {
		if anim.Type == "predefined" {
			modified = true
			continue
		}
		if anim.Type == "procedural" {
			if _, ok := builtins[anim.Slug]; !ok {
				modified = true
				continue
			}
		}
		cleanAnims = append(cleanAnims, anim)
	}
	db.state.Animations = cleanAnims

	for slug, builtin := range builtins {
		found := false
		for i, anim := range db.state.Animations {
			if anim.Slug == slug {
				found = true
				if anim.Type != builtin.Type || anim.Name != builtin.Name || anim.FrameDelayMs != builtin.FrameDelayMs {
					db.state.Animations[i].Type = builtin.Type
					db.state.Animations[i].Name = builtin.Name
					db.state.Animations[i].FrameDelayMs = builtin.FrameDelayMs
					modified = true
				}
				break
			}
		}
		if !found {
			db.state.Animations = append(db.state.Animations, builtin)
			modified = true
		}
	}

	if modified {
		_ = db.saveUnlocked()
	}
}

// save writes the state to disk (thread-safe)
func (db *DB) save() error {
	db.mu.Lock()
	defer db.mu.Unlock()
	return db.saveUnlocked()
}

// saveUnlocked writes state to disk without locking (must hold lock)
func (db *DB) saveUnlocked() error {
	data, err := json.MarshalIndent(db.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(db.dbPath, data, 0644)
}

// GetUsersCount returns the number of users registered
func (db *DB) GetUsersCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return len(db.state.Users)
}

// CreateUser registers a new user. The very first user is promoted to superadmin.
func (db *DB) CreateUser(username, passwordHash, salt string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if user already exists
	for _, u := range db.state.Users {
		if u.Username == username {
			return errors.New("kullanıcı zaten mevcut")
		}
	}

	role := "user"
	if len(db.state.Users) == 0 {
		role = "superadmin"
	}

	db.state.Users = append(db.state.Users, User{
		Username:     username,
		PasswordHash: passwordHash,
		Salt:         salt,
		Role:         role,
		CreatedAt:    time.Now(),
	})

	return db.saveUnlocked()
}

// GetUser retrieves a user by username
func (db *DB) GetUser(username string) (User, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, u := range db.state.Users {
		if u.Username == username {
			return u, true
		}
	}
	return User{}, false
}

// GetAnimations returns a copy of all animations
func (db *DB) GetAnimations() []Animation {
	db.mu.RLock()
	defer db.mu.RUnlock()

	anims := make([]Animation, len(db.state.Animations))
	copy(anims, db.state.Animations)
	return anims
}

// GetAnimation retrieves a single animation by slug
func (db *DB) GetAnimation(slug string) (Animation, bool) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	for _, anim := range db.state.Animations {
		if anim.Slug == slug {
			return anim, true
		}
	}
	return Animation{}, false
}

// CreateAnimation adds a new animation metadata
func (db *DB) CreateAnimation(slug, name, animType string, delayMs, framesCount int, username string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Validate slug uniqueness
	for _, anim := range db.state.Animations {
		if anim.Slug == slug {
			return errors.New("bu yol adına (slug) sahip başka bir yayın zaten mevcut")
		}
	}

	db.state.Animations = append(db.state.Animations, Animation{
		Slug:         slug,
		Name:         name,
		Type:         animType,
		FrameDelayMs: delayMs,
		FramesCount:  framesCount,
		CreatedAt:    time.Now(),
		CreatedBy:    username,
		RunCount:     0,
	})

	return db.saveUnlocked()
}

// DeleteAnimation deletes an animation
func (db *DB) DeleteAnimation(slug string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	index := -1
	for i, anim := range db.state.Animations {
		if anim.Slug == slug {
			if anim.Type == "procedural" {
				return errors.New("sistem animasyonları silinemez")
			}
			index = i
			break
		}
	}

	if index == -1 {
		return errors.New("animasyon bulunamadı")
	}

	// Delete animation file
	animFile := filepath.Join(filepath.Dir(db.dbPath), "animations", slug+".json")
	_ = os.Remove(animFile)

	// Remove from list
	db.state.Animations = append(db.state.Animations[:index], db.state.Animations[index+1:]...)

	return db.saveUnlocked()
}

// IncrementAnimationRunCount increments the run count of an animation
func (db *DB) IncrementAnimationRunCount(slug string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i, anim := range db.state.Animations {
		if anim.Slug == slug {
			db.state.Animations[i].RunCount++
			return db.saveUnlocked()
		}
	}
	return errors.New("animasyon bulunamadı")
}

// AddLog registers an access statistic record
func (db *DB) AddLog(slug, ip, ua, clientType string, duration float64) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	db.state.Logs = append(db.state.Logs, StatLog{
		Timestamp:       time.Now(),
		AnimationSlug:   slug,
		ClientIP:        ip,
		UserAgent:       ua,
		ClientType:      clientType,
		DurationSeconds: duration,
	})

	// Optional: cap logs to last 5000 entries to prevent file bloating
	if len(db.state.Logs) > 5000 {
		db.state.Logs = db.state.Logs[len(db.state.Logs)-5000:]
	}

	return db.saveUnlocked()
}

// GetLogs returns the last N logs
func (db *DB) GetLogs(limit int) []StatLog {
	db.mu.RLock()
	defer db.mu.RUnlock()

	logsCount := len(db.state.Logs)
	if logsCount == 0 {
		return []StatLog{}
	}

	if limit > logsCount {
		limit = logsCount
	}

	result := make([]StatLog, limit)
	for i := 0; i < limit; i++ {
		result[i] = db.state.Logs[logsCount-1-i]
	}
	return result
}

// GetTotalRunCount returns aggregate run count of all animations
func (db *DB) GetTotalRunCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	total := 0
	for _, anim := range db.state.Animations {
		total += anim.RunCount
	}
	return total
}

// GetUniqueIPsCount returns the number of unique client IPs in logs
func (db *DB) GetUniqueIPsCount() int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	uniqueIPs := make(map[string]struct{})
	for _, log := range db.state.Logs {
		uniqueIPs[log.ClientIP] = struct{}{}
	}
	return len(uniqueIPs)
}

// GetClientTypeCounts returns count of requests grouped by curl, wget, browser
func (db *DB) GetClientTypeCounts() map[string]int {
	db.mu.RLock()
	defer db.mu.RUnlock()

	counts := map[string]int{
		"curl":    0,
		"wget":    0,
		"browser": 0,
	}

	for _, log := range db.state.Logs {
		counts[log.ClientType]++
	}
	return counts
}
