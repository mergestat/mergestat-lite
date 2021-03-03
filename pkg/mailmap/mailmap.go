package mailmap

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type mailmap struct {
	filepath string
	userMap  map[string]string
}

func NewMailmap(filepath string) *mailmap {
	users := make(map[string]string)
	filepath += "/.mailmap"
	file, err := os.Open(filepath)
	if err != nil {
		fmt.Printf("repo does not contain mailmap at %s\n", filepath)
		return &mailmap{filepath: filepath, userMap: users}
	}
	buff := bufio.NewReader(file)
	for {
		line, _, err := buff.ReadLine()
		if err != nil {
			break
		}

		sects := strings.Split(string(line), "> ")
		if len(sects) > 1 {
			desired := strings.Split(sects[0], " <")
			desiredName := strings.TrimSpace(desired[0])
			desiredEmail := strings.Trim(desired[1], "<>")
			for x := 1; x < len(sects); x++ {
				new := strings.Split(sects[x], " <")
				if len(new) == 1 {
					email := strings.Trim(new[0], "<>")
					users[email] = desiredEmail
				} else {
					name := strings.TrimSpace(new[0])
					users[name] = desiredName
					email := strings.Trim(new[1], "<>")
					users[email] = desiredEmail
				}
			}
		}
	}
	return &mailmap{filepath: filepath, userMap: users}
}
func (m *mailmap) UseMailmap(user string) string {
	//println(filepath, user)

	_, ok := m.userMap[user]
	if ok {
		//println(user, users[user])
		return m.userMap[user]
	} else {
		return user
	}
}
