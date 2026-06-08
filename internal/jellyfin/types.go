package jellyfin

import (
	"time"
)

// Session represents a Jellyfin client session.
// This is what the Jellyfin for WebOS client expects.
type Session struct {
	Id                    string     `json:"Id"`
	UserId                string     `json:"UserId"`
	UserName              string     `json:"UserName"`
	Client                string     `json:"Client"`               // "Jellyfin for WebOS"
	DeviceName            string     `json:"DeviceName"`           // "LG TV"
	DeviceId              string     `json:"DeviceId"`
	ApplicationVersion    string     `json:"ApplicationVersion"`
	SupportsRemoteControl bool       `json:"SupportsRemoteControl"` // MUST be true
	SupportsMediaControl  bool       `json:"SupportsMediaControl"`
	PlayState             *PlayState `json:"PlayState,omitempty"`
	LastActivityDate      string     `json:"LastActivityDate"`
}

// PlayState represents the current playback state of a session.
type PlayState struct {
	IsPaused            bool   `json:"IsPaused"`
	PositionTicks        int64  `json:"PositionTicks"`
	PlayMethod           string `json:"PlayMethod"`
	MediaSourceId        string `json:"MediaSourceId"`
	ItemId               string `json:"ItemId"`
	AudioStreamIndex     *int   `json:"AudioStreamIndex,omitempty"`
	SubtitleStreamIndex  *int   `json:"SubtitleStreamIndex,omitempty"`
	CanSeek              bool   `json:"CanSeek"`
	IsMuted              bool   `json:"IsMuted"`
	VolumeLevel          int    `json:"VolumeLevel"`
	RepeatMode           string `json:"RepeatMode"`
}

// WSMessage represents a WebSocket message.
type WSMessage struct {
	MessageType string         `json:"MessageType"`
	Data        map[string]any `json:"Data,omitempty"`
}

// SessionChangeEvent is sent when a session changes.
type SessionChangeEvent struct {
	Session Session `json:"Session"`
}

// PlaybackProgressEvent is sent for playback progress updates.
type PlaybackProgressEvent struct {
	ItemId           string `json:"ItemId"`
	MediaSourceId    string `json:"MediaSourceId"`
	PositionTicks    int64  `json:"PositionTicks"`
	IsPaused         bool   `json:"IsPaused"`
	PlaySessionId    string `json:"PlaySessionId"`
	VolumeLevel      int    `json:"VolumeLevel"`
	IsMuted          bool   `json:"IsMuted"`
	PlayMethod       string `json:"PlayMethod"`
	AudioStreamIndex *int   `json:"AudioStreamIndex,omitempty"`
	SubtitleStreamIndex *int `json:"SubtitleStreamIndex,omitempty"`
}

// IdentityMessage is sent by client to identify itself.
type IdentityMessage struct {
	MessageType  string `json:"MessageType"`
	Client       string `json:"Client"`
	Device       string `json:"Device"`
	DeviceId     string `json:"DeviceId"`
	Version      string `json:"Version"`
	Token        string `json:"Token"`
	Capabilities []string `json:"Capabilities,omitempty"`
}

// HeartbeatMessage is sent periodically to keep connection alive.
type HeartbeatMessage struct {
	MessageType string `json:"MessageType"`
	Timestamp   string `json:"Timestamp"`
}

// WebSocket message types
const (
	WSMsgTypeIdentity        = "Identity"
	WSMsgTypeHeartbeat       = "Heartbeat"
	WSMsgTypeSessionChange   = "SessionChange"
	WSMsgTypePlaybackProgress = "PlaybackProgress"
	WSMsgTypePlaybackStart   = "PlaybackStart"
	WSMsgTypePlaybackStop    = "PlaybackStop"
)

// SessionKey is used for session storage.
type SessionKey struct {
	DeviceId string
	UserId   string
}

// SessionStore is an in-memory store for active sessions.
type SessionStore struct {
	sessions map[string]*Session // sessionId -> Session
}

// NewSessionStore creates a new session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{
		sessions: make(map[string]*Session),
	}
}

// Get returns a session by ID.
func (s *SessionStore) Get(id string) (*Session, bool) {
	session, ok := s.sessions[id]
	return session, ok
}

// Set stores a session.
func (s *SessionStore) Set(session *Session) {
	s.sessions[session.Id] = session
}

// Delete removes a session.
func (s *SessionStore) Delete(id string) {
	delete(s.sessions, id)
}

// List returns all sessions.
func (s *SessionStore) List() []Session {
	sessions := make([]Session, 0, len(s.sessions))
	for _, s := range s.sessions {
		sessions = append(sessions, *s)
	}
	return sessions
}

// FindWebOSSession finds the first WebOS session that supports remote control.
func (s *SessionStore) FindWebOSSession() *Session {
	for _, session := range s.sessions {
		if (session.Client == "Jellyfin for WebOS" || session.Client == "Jellyfin WebOS") &&
			session.SupportsRemoteControl {
			return session
		}
	}
	return nil
}

// SessionConfig holds configuration for the Jellyfin session manager.
type SessionConfig struct {
	ServerName        string
	ServerVersion     string
	DeviceId          string
	HeartbeatInterval time.Duration
	SessionTimeout    time.Duration
}