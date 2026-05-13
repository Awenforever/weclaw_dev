package runtime_state

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

type State struct {
	DefaultProfile string                 `json:"default_profile,omitempty"`
	Sessions       map[string]SessionHint `json:"sessions,omitempty"`
	UpdatedUnix    int64                  `json:"updated_unix,omitempty"`
}

type SessionHint struct {
	Profile     string `json:"profile,omitempty"`
	UserID      string `json:"user_id,omitempty"`
	SessionID   string `json:"session_id,omitempty"`
	UpdatedUnix int64  `json:"updated_unix,omitempty"`
}

var stateMu sync.Mutex

func Dir() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return "."
	}
	return filepath.Join(home, ".weclaw")
}

func Path() string {
	return filepath.Join(Dir(), "runtime-state.json")
}

func LogPath() string {
	return filepath.Join(Dir(), "weclaw.log")
}

func Load() (State, error) {
	stateMu.Lock()
	defer stateMu.Unlock()
	return loadUnlocked()
}

func SetDefaultProfile(profile string) error {
	profile = strings.TrimSpace(profile)
	if profile == "" {
		return nil
	}

	stateMu.Lock()
	defer stateMu.Unlock()

	state, err := loadUnlocked()
	if err != nil {
		return err
	}
	state.DefaultProfile = profile
	state.UpdatedUnix = time.Now().Unix()
	return saveUnlocked(state)
}

func UpsertSession(profile, userID, sessionID string) error {
	profile = strings.TrimSpace(profile)
	userID = strings.TrimSpace(userID)
	sessionID = strings.TrimSpace(sessionID)
	if profile == "" || userID == "" || sessionID == "" {
		return nil
	}

	stateMu.Lock()
	defer stateMu.Unlock()

	state, err := loadUnlocked()
	if err != nil {
		return err
	}
	if state.Sessions == nil {
		state.Sessions = make(map[string]SessionHint)
	}
	if state.DefaultProfile == "" {
		state.DefaultProfile = profile
	}
	now := time.Now().Unix()
	key := profile + "|" + userID
	state.Sessions[key] = SessionHint{
		Profile:     profile,
		UserID:      userID,
		SessionID:   sessionID,
		UpdatedUnix: now,
	}
	state.UpdatedUnix = now
	return saveUnlocked(state)
}

func MostRecentSessionForDefaultProfile() (profile string, sessionID string, ok bool) {
	state, err := Load()
	if err != nil {
		return "", "", false
	}

	profile = strings.TrimSpace(state.DefaultProfile)
	best, ok := mostRecentSessionMatching(state, profile)
	if !ok {
		return "", "", false
	}
	if profile == "" {
		profile = best.Profile
	}
	return profile, best.SessionID, true
}

func MostRecentSessionForProfile(profile string) (SessionHint, bool) {
	state, err := Load()
	if err != nil {
		return SessionHint{}, false
	}
	return mostRecentSessionMatching(state, strings.TrimSpace(profile))
}

func mostRecentSessionMatching(state State, profile string) (SessionHint, bool) {
	var best SessionHint
	for _, hint := range state.Sessions {
		if strings.TrimSpace(hint.SessionID) == "" || strings.TrimSpace(hint.Profile) == "" {
			continue
		}
		if profile != "" && hint.Profile != profile {
			continue
		}
		if best.SessionID == "" || hint.UpdatedUnix >= best.UpdatedUnix {
			best = hint
		}
	}
	if best.SessionID == "" {
		return SessionHint{}, false
	}
	return best, true
}

func MostRecentLogThreadForPIDs(profile string, pids []int) (SessionHint, bool) {
	data, err := os.ReadFile(LogPath())
	if err != nil {
		return SessionHint{}, false
	}
	allowed := make(map[int]bool, len(pids))
	for _, pid := range pids {
		if pid > 0 {
			allowed[pid] = true
		}
	}

	re := regexp.MustCompile(`\[acp\] (?:new thread created|reusing thread) \(pid=([0-9]+), thread=([^,]+), conversation=([^)]+)\)`)
	var best SessionHint
	for _, line := range strings.Split(string(data), "\n") {
		match := re.FindStringSubmatch(line)
		if len(match) != 4 {
			continue
		}
		pid, err := strconv.Atoi(match[1])
		if err != nil {
			continue
		}
		if len(allowed) > 0 && !allowed[pid] {
			continue
		}
		threadID := strings.TrimSpace(match[2])
		userID := strings.TrimSpace(match[3])
		if threadID == "" || userID == "" {
			continue
		}
		best = SessionHint{
			Profile:   strings.TrimSpace(profile),
			UserID:    userID,
			SessionID: threadID,
		}
	}
	if best.SessionID == "" {
		return SessionHint{}, false
	}
	return best, true
}

func loadUnlocked() (State, error) {
	data, err := os.ReadFile(Path())
	if err != nil {
		if os.IsNotExist(err) {
			return State{}, nil
		}
		return State{}, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return State{}, nil
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, err
	}
	if state.Sessions == nil {
		state.Sessions = make(map[string]SessionHint)
	}
	return state, nil
}

func saveUnlocked(state State) error {
	if err := os.MkdirAll(Dir(), 0o700); err != nil {
		return err
	}
	if state.Sessions == nil {
		state.Sessions = make(map[string]SessionHint)
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	target := Path()
	tmp, err := os.CreateTemp(filepath.Dir(target), ".runtime-state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, target)
}
