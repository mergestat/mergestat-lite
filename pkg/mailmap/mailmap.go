// Package mailmap implements a basic git mailmap parser. See this page: https://git-scm.com/docs/gitmailmap for additional context.
package mailmap

import "strings"

type NameAndEmail struct {
	Name  string
	Email string
}

// MailMap maps names and emails found in commits to "proper" names and emails.
// The map key is the "proper" pair, coresponding to a list of pairs to be matched.
type MailMap map[NameAndEmail][]NameAndEmail

// Parse takes an input mailmap string and parses it into a MailMap struct
func Parse(input string) (MailMap, error) {
	// someone smarter can and should probably implement a better approach here.
	// this implementation uses a bunch of string splitting on the characters: < and >
	// and might have bugs for weird edge cases. See the tests for what's supported.
	// See also: https://github.com/libgit2/libgit2/blob/main/src/mailmap.c
	out := make(MailMap)
	for _, line := range strings.Split(input, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") { // ignore comments
			continue
		}

		var properName, properEmail, commitName, commitEmail string
		s := strings.FieldsFunc(line, func(r rune) bool {
			return r == '<' || r == '>'
		})
		// Proper Name <commit@email>
		// Proper Name <proper@email> <commit@email>
		// Proper Name <proper@email> Commit Name <commit@email>
		switch len(s) {
		case 2:
			properName = strings.TrimSpace(s[0])
			commitEmail = strings.TrimSpace(s[1])
		case 4:
			properName = strings.TrimSpace(s[0])
			properEmail = strings.TrimSpace(s[1])
			commitName = strings.TrimSpace(s[2])
			commitEmail = strings.TrimSpace(s[3])
		default:
			continue
		}

		if properName == "" && properEmail == "" {
			continue
		}

		proper := NameAndEmail{Name: properName, Email: properEmail}
		if _, ok := out[proper]; !ok {
			out[proper] = make([]NameAndEmail, 0)
		}

		out[proper] = append(out[proper], NameAndEmail{Name: commitName, Email: commitEmail})
	}

	return out, nil
}

// Lookup receives a name/email pair and finds the first proper name/email pair
func (mm MailMap) Lookup(commitLookup NameAndEmail) *NameAndEmail {
	for proper, commits := range mm {
		for _, commit := range commits {
			// case insensitive match
			if strings.EqualFold(commit.Name, commitLookup.Name) && strings.EqualFold(commit.Email, commit.Email) {
				return &proper
			}
		}
	}
	return nil
}
