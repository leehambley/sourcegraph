package gitlab

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"strconv"
	"time"

	"github.com/sourcegraph/sourcegraph/cmd/frontend/authz"
	"github.com/sourcegraph/sourcegraph/cmd/frontend/types"
	"github.com/sourcegraph/sourcegraph/pkg/api"
	"github.com/sourcegraph/sourcegraph/pkg/extsvc"
	"github.com/sourcegraph/sourcegraph/pkg/extsvc/gitlab"
	"github.com/sourcegraph/sourcegraph/pkg/rcache"
	log15 "gopkg.in/inconshreveable/log15.v2"
)

// TODO: this should replace the old sudo-token-based auth method

type GitLabOAuthAuthzProvider struct {
	client    *gitlab.Client
	clientURL *url.URL
	codeHost  *gitlab.CodeHost
	cache     pcache
	cacheTTL  time.Duration
}

type GitLabOAuthAuthzProviderOp struct {
	// BaseURL is the URL of the GitLab instance.
	BaseURL *url.URL

	// CacheTTL is the TTL of cached permissions lists from the GitLab API.
	CacheTTL time.Duration

	// MockCache, if non-nil, replaces the default Redis-based cache with the supplied cache mock.
	// Should only be used in tests.
	MockCache pcache
}

func NewProvider(op GitLabOAuthAuthzProviderOp) *GitLabOAuthAuthzProvider {
	p := &GitLabOAuthAuthzProvider{
		client:    gitlab.NewClient(op.BaseURL, "", "", nil),
		clientURL: op.BaseURL,
		codeHost:  gitlab.NewCodeHost(op.BaseURL),
		cache:     op.MockCache,
		cacheTTL:  op.CacheTTL,
	}
	if p.cache == nil {
		p.cache = rcache.NewWithTTL(fmt.Sprintf("gitlabAuthz:%s", op.BaseURL.String()), int(math.Ceil(op.CacheTTL.Seconds())))
	}
	return p
}

func (p *GitLabOAuthAuthzProvider) Validate() (problems []string) {
	// TODO(beyang)
	return nil
}

func (p *GitLabOAuthAuthzProvider) ServiceID() string {
	return p.codeHost.ServiceID()
}

func (p *GitLabOAuthAuthzProvider) ServiceType() string {
	return p.codeHost.ServiceType()
}

func (p *GitLabOAuthAuthzProvider) RepoPerms(ctx context.Context, account *extsvc.ExternalAccount, repos map[authz.Repo]struct{}) (map[api.RepoName]map[authz.Perm]bool, error) {
	accountID := "" // empty means public / unauthenticated to the code host
	if account != nil && account.ServiceID == p.codeHost.ServiceID() && account.ServiceType == p.codeHost.ServiceType() {
		accountID = account.AccountID
	}

	myRepos, _ := p.Repos(ctx, repos)
	var accessibleRepos map[int]struct{}
	if r, exists := p.getCachedAccessList(accountID); exists {
		accessibleRepos = r
	} else {
		var err error

		_, tok, err := gitlab.GetExternalAccountData(account.AccountData)
		if err != nil {
			return nil, err
		}

		// NEXT

		accessibleRepos, err = p.fetchUserAccessList(ctx, accountID)
		if err != nil {
			return nil, err
		}

		accessibleReposB, err := json.Marshal(cacheVal{
			ProjIDs: accessibleRepos,
			TTL:     p.cacheTTL,
		})
		if err != nil {
			return nil, err
		}
		p.cache.Set(accountID, accessibleReposB)
	}

	perms := make(map[api.RepoName]map[authz.Perm]bool)
	for repo := range myRepos {
		perms[repo.RepoName] = map[authz.Perm]bool{}

		projID, err := strconv.Atoi(repo.ExternalRepoSpec.ID)
		if err != nil {
			log15.Warn("couldn't parse GitLab proj ID as int while computing permissions", "id", repo.ExternalRepoSpec.ID)
			continue
		}
		_, isAccessible := accessibleRepos[projID]
		if !isAccessible {
			continue
		}
		perms[repo.RepoName][authz.Read] = true
	}

	return perms, nil
}

func (p *GitLabOAuthAuthzProvider) Repos(ctx context.Context, repos map[authz.Repo]struct{}) (mine map[authz.Repo]struct{}, others map[authz.Repo]struct{}) {
	return authz.GetCodeHostRepos(p.codeHost, repos)
}

func (p *GitLabOAuthAuthzProvider) FetchAccount(ctx context.Context, user *types.User, current []*extsvc.ExternalAccount) (mine *extsvc.ExternalAccount, err error) {
	return nil, nil
}

// fetchUserAccessList fetches the list of project IDs that are readable to a user from the GitLab API.
func (p *GitLabOAuthAuthzProvider) fetchUserAccessList(ctx context.Context, glUserID string) (map[int]struct{}, error) {
	q := make(url.Values)
	if glUserID != "" {
		q.Add("sudo", glUserID)
	} else {
		q.Add("visibility", "public")
	}
	q.Add("per_page", "100")

	projIDs := make(map[int]struct{})
	var iters = 0
	var pageURL = "projects?" + q.Encode()
	for {
		if iters >= 100 && iters%100 == 0 {
			log15.Warn("Excessively many GitLab API requests to fetch complete user authz list", "iters", iters, "gitlabUserID", glUserID, "host", p.clientURL.String())
		}

		projs, nextPageURL, err := p.client.ListProjects(ctx, pageURL)
		if err != nil {
			return nil, err
		}
		for _, proj := range projs {
			projIDs[proj.ID] = struct{}{}
		}

		if nextPageURL == nil {
			break
		}
		pageURL = *nextPageURL
		iters++
	}
	return projIDs, nil
}
