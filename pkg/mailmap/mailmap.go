package mailmap

import (
	"encoding/json"
	"strings"
	//git "github.com/libgit2/git2go/v31"
)

// type mailmap struct {
//  	filepath string
//  	userMap  map[string]string
//  }
func Mailmap(contents string) string {
	mailmapContents := strings.Split(contents, "\n")
	users := make(map[string]string)
	for _, c := range mailmapContents {
		if !(strings.HasPrefix(c, "#")) {
			line := c
			//println(c)
			sects := strings.Split(line, "> ")
			//println(strings.Join(sects, " : "))
			if len(sects) > 1 {
				// if this is email only for the first
				// if sects[0][0] == '<' {

				// }
				desired := strings.Split(sects[0], " <")
				//println(strings.Join(desired, " : "))
				var (
					desiredName  string
					desiredEmail string
				)
				if len(desired) > 1 {
					desiredName = strings.TrimSpace(desired[0])
					desiredEmail = strings.Trim(desired[1], "<>")
				} else {
					desiredEmail = strings.Trim(desired[0], "<>")
				}
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
	}
	ret, _ := json.Marshal(users)
	return string(ret)
}

// func NewMailmap(filepath string) (*mailmap, error) {
// 	users := make(map[string]string)
// 	repo, err := git.OpenRepository(filepath)
// 	if err != nil {
// 		fmt.Printf("repo does cannot be opened at %s\n", filepath)
// 		return &mailmap{filepath: filepath, userMap: users}, err
// 	}
// 	head, err := repo.Head()
// 	if err != nil {
// 		panic(err)
// 	}
// 	commit, err := repo.LookupCommit(head.Target())
// 	if err != nil {
// 		fmt.Printf("getting head from repo generated error: %s\n", err.Error())
// 		return &mailmap{filepath: filepath, userMap: users}, err
// 	}
// 	tree, err := commit.Tree()
// 	if err != nil {
// 		fmt.Printf("getting tree from head generated error: %s", err.Error())
// 		return &mailmap{filepath: filepath, userMap: users}, err
// 	}
// 	var mlmpblob *git.Blob
// 	for i := 0; i < int(tree.EntryCount()); i++ {
// 		if strings.Contains(tree.EntryByIndex(uint64(i)).Name, ".mailmap") {
// 			mlmp := tree.EntryByIndex(uint64(i))
// 			mlmpobj, err := repo.Lookup(mlmp.Id)
// 			if err != nil {
// 				return nil, err
// 			}
// 			mlmpblob, err = mlmpobj.AsBlob()
// 			if err != nil {
// 				return nil, err
// 			}
// 			// break out of loop as soon as mailmap is found to avoid iterating over entire tree
// 			break
// 		}
// 	}
// 	if mlmpblob == nil {
// 		return &mailmap{filepath: filepath, userMap: users}, nil
// 	}
// 	s := string(mlmpblob.Contents())
// 	contents := strings.Split(s, "\n")
// 	for _, c := range contents {
// 		if !(strings.HasPrefix(c, "#")) {
// 			line := c
// 			//println(c)
// 			sects := strings.Split(line, "> ")
// 			//println(strings.Join(sects, " : "))
// 			if len(sects) > 1 {
// 				// if this is email only for the first
// 				// if sects[0][0] == '<' {

// 				// }
// 				desired := strings.Split(sects[0], " <")
// 				//println(strings.Join(desired, " : "))
// 				var (
// 					desiredName  string
// 					desiredEmail string
// 				)
// 				if len(desired) > 1 {
// 					desiredName = strings.TrimSpace(desired[0])
// 					desiredEmail = strings.Trim(desired[1], "<>")
// 				} else {
// 					desiredEmail = strings.Trim(desired[0], "<>")
// 				}
// 				for x := 1; x < len(sects); x++ {
// 					new := strings.Split(sects[x], " <")
// 					if len(new) == 1 {
// 						email := strings.Trim(new[0], "<>")
// 						users[email] = desiredEmail
// 					} else {
// 						name := strings.TrimSpace(new[0])
// 						users[name] = desiredName
// 						email := strings.Trim(new[1], "<>")
// 						users[email] = desiredEmail
// 					}
// 				}
// 			}
// 		}
// 	}
// 	return &mailmap{filepath: filepath, userMap: users}, nil
// }
// func (m *mailmap) UseMailmap(user string) string {
// 	//println(filepath, user)

// 	_, ok := m.userMap[user]
// 	if ok {
// 		//println(user, users[user])
// 		return m.userMap[user]
// 	} else {
// 		return user
// 	}
// }
