package gitlab

/*
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
	p := &GitLabAuthzProvider{
		client:    gitlab.NewClient(op.BaseURL, op.SudoToken, "", nil),
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

func (p *GitLabAuthzProvider) Validate() (problems []string) {
	// TODO(beyang)
	return nil
}

func (p *GitLabAuthzProvider) ServiceID() string {
	return p.codeHost.ServiceID()
}

func (p *GitLabAuthzProvider) ServiceType() string {
	return p.codeHost.ServiceType()
}

func (p *GitLabAuthzProvider) RepoPerms(ctx context.Context, account *extsvc.ExternalAccount, repos map[authz.Repo]struct{}) (map[api.RepoName]map[authz.Perm]bool, error) {
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
		// gitlab.GetExternalAccountData

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
*/
