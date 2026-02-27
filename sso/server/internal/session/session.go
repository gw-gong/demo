package session

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

type Store struct {
	path string
	mu   sync.RWMutex
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func sessionLine(sessionID, userID string, expire int64) string {
	return fmt.Sprintf("%s\t%s\t%d", sessionID, userID, expire)
}

func parseLine(line string) (sessionID, userID string, expire int64, ok bool) {
	parts := strings.SplitN(strings.TrimSpace(line), "\t", 3)
	if len(parts) < 3 {
		return "", "", 0, false
	}
	var exp int64
	if _, err := fmt.Sscanf(parts[2], "%d", &exp); err != nil {
		return "", "", 0, false
	}
	return parts[0], parts[1], exp, true
}

func (s *Store) Create(sessionID, userID string, expireAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.OpenFile(s.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(sessionLine(sessionID, userID, expireAt.Unix()) + "\n")
	return err
}

func (s *Store) Find(sessionID string) (userID string, ok bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	f, err := os.Open(s.path)
	if err != nil {
		return "", false
	}
	defer f.Close()
	now := time.Now().Unix()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		sid, uid, exp, ok := parseLine(sc.Text())
		if !ok || sid != sessionID {
			continue
		}
		if exp <= now {
			return "", false
		}
		return uid, true
	}
	return "", false
}

func (s *Store) Delete(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	f, err := os.Open(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	sc := bufio.NewScanner(f)
	var lines []string
	for sc.Scan() {
		line := sc.Text()
		sid, _, _, ok := parseLine(line)
		if ok && sid == sessionID {
			continue
		}
		lines = append(lines, line)
	}
	f.Close()
	return os.WriteFile(s.path, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}
