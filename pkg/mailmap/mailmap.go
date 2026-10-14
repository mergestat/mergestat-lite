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

// Parse takes an input mailmap string and parses it into a MailMap struct according to the gitmailmap specification.
func Parse(input string) (MailMap, error) {
	out := make(MailMap)
	for _, line := range strings.Split(input, "\n") {
		properName, properEmail, commitName, commitEmail, ok := parseLine(line)
		if !ok {
			continue
		}

		proper := NameAndEmail{Name: properName, Email: properEmail}
		commit := NameAndEmail{Name: commitName, Email: commitEmail}

		// Replace any previous mapping matching this commit author to allow later entries to override
		for p, commits := range out {
			filtered := commits[:0]
			for _, c := range commits {
				if !(strings.EqualFold(c.Email, commit.Email) && strings.EqualFold(c.Name, commit.Name)) {
					filtered = append(filtered, c)
				}
			}
			out[p] = filtered
		}

		out[proper] = append(out[proper], commit)
	}

	return out, nil
}

// parseLine parses a single line of a .mailmap file according to the gitmailmap specification.
// Supported syntax forms:
//   Proper Name <commit@email>
//   <proper@email> <commit@email>
//   Proper Name <proper@email> <commit@email>
//   Proper Name <proper@email> Commit Name <commit@email>
//   <proper@email> Commit Name <commit@email>
// Lines may include comments starting with '#' and arbitrary whitespace.
func parseLine(line string) (properName, properEmail, commitName, commitEmail string, ok bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return "", "", "", "", false
	}

	firstLt := strings.IndexByte(line, '<')
	if firstLt < 0 {
		return "", "", "", "", false
	}

	// Comment before the first '<' makes the entire line a comment
	if strings.Contains(line[:firstLt], "#") {
		return "", "", "", "", false
	}

	properName = strings.TrimSpace(line[:firstLt])

	rest := line[firstLt+1:]
	firstGt := strings.IndexByte(rest, '>')
	if firstGt < 0 {
		return "", "", "", "", false
	}

	email1 := strings.TrimSpace(rest[:firstGt])
	afterFirstGt := rest[firstGt+1:]

	secondLt := strings.IndexByte(afterFirstGt, '<')
	commentIdx := strings.IndexByte(afterFirstGt, '#')
	if secondLt < 0 || (commentIdx >= 0 && commentIdx < secondLt) {
		// Single email on line: Proper Name <commit@email>
		remainder := afterFirstGt
		if commentIdx >= 0 {
			remainder = afterFirstGt[:commentIdx]
		}
		if strings.TrimSpace(remainder) != "" {
			return "", "", "", "", false
		}

		if email1 == "" || properName == "" {
			return "", "", "", "", false
		}

		return properName, "", "", email1, true
	}

	// Two emails on line: [Proper Name] <proper@email> [Commit Name] <commit@email>
	commitName = strings.TrimSpace(afterFirstGt[:secondLt])

	afterSecondLt := afterFirstGt[secondLt+1:]
	secondGt := strings.IndexByte(afterSecondLt, '>')
	if secondGt < 0 {
		return "", "", "", "", false
	}

	commitEmail = strings.TrimSpace(afterSecondLt[:secondGt])
	properEmail = email1

	remainder := afterSecondLt[secondGt+1:]
	commentIdx2 := strings.IndexByte(remainder, '#')
	if commentIdx2 >= 0 {
		remainder = remainder[:commentIdx2]
	}
	if strings.TrimSpace(remainder) != "" {
		return "", "", "", "", false
	}

	if commitEmail == "" || (properName == "" && properEmail == "") {
		return "", "", "", "", false
	}

	return properName, properEmail, commitName, commitEmail, true
}

// Lookup receives a name/email pair and finds the proper name/email pair.
// It prioritizes specific matches (matching both commit name and commit email) over
// generic matches (matching commit email with no specified commit name).
// If a proper name is mapped but proper email is omitted, the original commit email is preserved.
// If a proper email is mapped but proper name is omitted, the original commit name is preserved.
func (mm MailMap) Lookup(commitLookup NameAndEmail) NameAndEmail {
	var fallbackProper *NameAndEmail

	for proper, commits := range mm {
		for _, commit := range commits {
			if !strings.EqualFold(commit.Email, commitLookup.Email) {
				continue
			}

			// Specific match: commit.Name is non-empty and matches
			if commit.Name != "" && strings.EqualFold(commit.Name, commitLookup.Name) {
				result := commitLookup
				if proper.Name != "" {
					result.Name = proper.Name
				}
				if proper.Email != "" {
					result.Email = proper.Email
				}
				return result
			}

			// Generic match: commit.Name is empty (matches any commit name)
			if commit.Name == "" && fallbackProper == nil {
				properCopy := proper
				fallbackProper = &properCopy
			}
		}
	}

	if fallbackProper != nil {
		result := commitLookup
		if fallbackProper.Name != "" {
			result.Name = fallbackProper.Name
		}
		if fallbackProper.Email != "" {
			result.Email = fallbackProper.Email
		}
		return result
	}

	return commitLookup
}
