package users

import (
	"bufio"
	"os"
	"strings"
)

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
}

type Entry struct {
	User     User
	Password string
}

// Load 从 users.txt 加载，每行 username\tpassword\tname
func Load(path string) ([]Entry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var out []Entry
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "\t", 3)
		if len(parts) < 2 {
			continue
		}
		username := parts[0]
		password := parts[1]
		name := username
		if len(parts) >= 3 {
			name = parts[2]
		}
		out = append(out, Entry{
			User:     User{ID: username, Username: username, Name: name},
			Password: password,
		})
	}
	return out, sc.Err()
}

func Validate(users []Entry, username, password string) *User {
	for i := range users {
		if users[i].User.Username == username && users[i].Password == password {
			return &users[i].User
		}
	}
	return nil
}

func FindByID(users []Entry, id string) *User {
	for i := range users {
		if users[i].User.ID == id {
			return &users[i].User
		}
	}
	return nil
}
