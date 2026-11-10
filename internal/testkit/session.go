package testkit

import "sync"

// SessionState is connection-local state for the stateful era.
// 2026-07-28 fixtures keep this empty: catalogs do not vary by connection.
type SessionState struct {
	sessionID    string
	lastEventID  string
	mu           sync.Mutex
	getCount     int
	discoverSeen int
	initialized  bool
	tornDown     bool
}

func (s *SessionState) markInitialized() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.initialized = true
}

func (s *SessionState) ready(require bool) bool {
	if !require {
		return true
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.initialized
}

func (s *SessionState) setSessionID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.sessionID = id
}

func (s *SessionState) currentSessionID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.sessionID
}

func (s *SessionState) tearDown() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tornDown = true
}

func (s *SessionState) isTornDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tornDown
}

func (s *SessionState) noteGET(lastEventID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.getCount++
	s.lastEventID = lastEventID
	return s.getCount
}

func (s *SessionState) GETCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.getCount
}

func (s *SessionState) SeenLastEventID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastEventID
}

func (s *SessionState) nextDiscover() int {
	if s == nil {
		return 1
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.discoverSeen++
	return s.discoverSeen
}
