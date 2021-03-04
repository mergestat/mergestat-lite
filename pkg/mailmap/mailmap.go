package mailmap

import (
	"fmt"
	"strings"

	git "github.com/libgit2/git2go/v31"
)

type mailmap struct {
	filepath string
	userMap  map[string]string
}

func NewMailmap(filepath string) *mailmap {
	users := make(map[string]string)
	repo, err := git.OpenRepository(filepath)
	if err != nil {
		fmt.Printf("repo does cannot be opened at %s\n", filepath)
		return &mailmap{filepath: filepath, userMap: users}
	}
	head, err := repo.Head()
	if err != nil {
		panic(err)
	}
	commit, err := repo.LookupCommit(head.Target())
	if err != nil {
		fmt.Printf("getting head from repo generated error: %s\n", err.Error())
		return &mailmap{filepath: filepath, userMap: users}
	}
	tree, err := commit.Tree()
	if err != nil {
		fmt.Printf("getting tree from head generated error: %s", err.Error())
		return &mailmap{filepath: filepath, userMap: users}
	}
	var mlmpblob *git.Blob
	for i := 0; i < int(tree.EntryCount()); i++ {
		if strings.Contains(tree.EntryByIndex(uint64(i)).Name, ".mailmap") {
			mlmp := tree.EntryByIndex(uint64(i))
			mlmpobj, err := repo.Lookup(mlmp.Id)
			if err != nil {
				panic(err)
			}
			mlmpblob, err = mlmpobj.AsBlob()
			if err != nil {
				panic(err)
			}
			// break out of loop as soon as mailmap is found to avoid iterating over entire tree
			break
		}
	}
	for _, c := range mlmpblob.Contents() {
		line := c

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
