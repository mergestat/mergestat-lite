package github

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/mergestat/mergestat/extensions/options"
	"github.com/shurcooL/githubv4"
	"go.riyazali.net/sqlite"
)

type repoInfo struct {
	opts *Options
}

func (r *repoInfo) Args() int           { return -1 }
func (r *repoInfo) Deterministic() bool { return false }
func (r *repoInfo) Apply(ctx *sqlite.Context, values ...sqlite.Value) {
	err := r.opts.RateLimiter.Wait(context.Background())
	if err != nil {
		ctx.ResultError(err)
		return
	}

	// TODO(patrickdevivo) seems silly to unmarshal the graphql query into a struct
	// and then immediately re-marshal it into json for output...should probably make this simpler
	// by using the graphQL query directly and returning the []byte result directly
	var repoInfoQuery struct {
		RateLimit  *options.GitHubRateLimitResponse
		Repository struct {
			CreatedAt        time.Time
			DefaultBranchRef struct {
				Name   string
				Prefix string
			}
			Description string
			DiskUsage   int
			ForkCount   int
			HomepageUrl string
			IsArchived  bool
			IsDisabled  bool
			IsFork      bool
			IsMirror    bool
			IsPrivate   bool
			Issues      struct {
				TotalCount int
			}
			LatestRelease struct {
				Author struct {
					Login string
				}
				CreatedAt   githubv4.DateTime
				Name        string
				PublishedAt githubv4.DateTime
			}
			LicenseInfo struct {
				Key      string
				Name     string
				Nickname string
			}
			Name              string
			OpenGraphImageUrl githubv4.URI
			PrimaryLanguage   struct {
				Name string
			}
			PullRequests struct {
				TotalCount int
			}
			PushedAt time.Time
			Releases struct {
				TotalCount int
			}
			StargazerCount int
			Topics         struct {
				Nodes []struct {
					Topic struct {
						Name string
					}
				}
			} `graphql:"repositoryTopics(first: 10)"`
			UpdatedAt time.Time
			Watchers  struct {
				TotalCount int
			}
		} `graphql:"repository(owner: $owner, name: $name)"`
	}

	var owner, name string
	switch len(values) {
	case 0:
		ctx.ResultError(errors.New("need to supply a repo"))
		return
	case 1:
		split_string := strings.Split(values[0].Text(), "/")
		if len(split_string) != 2 {
			ctx.ResultError(errors.New("invalid repo name, must be of format owner/name"))
			return
		}
		owner = split_string[0]
		name = split_string[1]
	default:
		owner = values[0].Text()
		name = values[1].Text()
	}

	r.opts.GitHubPreRequestHook()

	l := r.opts.Logger.With().Str("owner", owner).Str("name", name).Logger()
	l.Info().Msgf("fetching repo info from GitHub for: %s/%s", owner, name)

	variables := map[string]interface{}{
		"owner": githubv4.String(owner),
		"name":  githubv4.String(name),
	}
	err = r.opts.Client().Query(context.Background(), &repoInfoQuery, variables)

	r.opts.GitHubPostRequestHook()

	if err != nil {
		ctx.ResultError(err)
		return
	}

	r.opts.RateLimitHandler(repoInfoQuery.RateLimit)

	repo := repoInfoQuery.Repository

	toJSON := map[string]interface{}{
		"createdAt":         repo.CreatedAt,
		"defaultBranchName": repo.DefaultBranchRef.Name,
		"description":       repo.Description,
		"diskUsage":         repo.DiskUsage,
		"forkCount":         repo.ForkCount,
		"homepageURL":       repo.HomepageUrl,
		"isArchived":        repo.IsArchived,
		"isDisabled":        repo.IsDisabled,
		"isMirror":          repo.IsMirror,
		"isPrivate":         repo.IsPrivate,
		"totalIssuesCount":  repo.Issues.TotalCount,
		"latestRelease": map[string]interface{}{
			"authorLogin": repo.LatestRelease.Author.Login,
			"createdAt":   repo.LatestRelease.CreatedAt,
			"name":        repo.LatestRelease.Name,
			"publishedAt": repo.LatestRelease.PublishedAt,
		},
		"license": map[string]interface{}{
			"key":      repo.LicenseInfo.Key,
			"name":     repo.LicenseInfo.Name,
			"nickname": repo.LicenseInfo.Nickname,
		},
		"name":              repo.Name,
		"openGraphImageURL": repo.OpenGraphImageUrl,
		"primaryLanguage":   repo.PrimaryLanguage.Name,
		"pushedAt":          repo.PushedAt,
		"releasesCount":     repo.Releases.TotalCount,
		"stargazersCount":   repo.StargazerCount,
		"updatedAt":         repo.UpdatedAt,
		"watchersCount":     repo.Watchers.TotalCount,
	}

	topics := make([]string, len(repo.Topics.Nodes))
	for i, t := range repo.Topics.Nodes {
		topics[i] = t.Topic.Name
	}
	toJSON["topics"] = topics

	var out []byte
	if out, err = json.Marshal(toJSON); err != nil {
		ctx.ResultError(err)
		return
	}

	ctx.ResultText(string(out))
}

func NewRepoInfoFunc(opts *Options) sqlite.Function {
	return &repoInfo{opts}
}
