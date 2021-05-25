package gitproviders

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/dnaeon/go-vcr/cassette"

	"github.com/dnaeon/go-vcr/recorder"

	"github.com/fluxcd/go-git-providers/github"
	"github.com/fluxcd/go-git-providers/gitlab"
	"github.com/stretchr/testify/assert"

	"github.com/fluxcd/go-git-providers/gitprovider"
)

var cacheGithubRecorder, cacheGitlabRecorder *recorder.Recorder

var (
	GithubOrgTestName  = "weaveworks"
	GithubUserTestName = "bot"
	GitlabOrgTestName  = "weaveworks"
	GitlabUserTestName = "bot"
)

func SetRecorder(recorder *recorder.Recorder) gitprovider.ChainableRoundTripperFunc {
	return func(transport http.RoundTripper) http.RoundTripper {
		recorder.SetTransport(transport)
		return recorder
	}
}

type accounts struct {
	GithubOrgName  string
	GithubUserName string
	GitlabOrgName  string
	GitlabUserName string
}

func NewRecorder(provider string, accounts *accounts) (*recorder.Recorder, error) {
	r, err := recorder.New(fmt.Sprintf("./cache/%s", provider))
	if err != nil {
		return nil, err
	}

	r.SetMatcher(func(r *http.Request, i cassette.Request) bool {
		if accounts.GithubOrgName != GithubOrgTestName {
			r.URL, _ = url.Parse(strings.Replace(r.URL.String(), accounts.GithubOrgName, GithubOrgTestName, -1))
			r.URL, _ = url.Parse(strings.Replace(r.URL.String(), accounts.GithubUserName, GithubUserTestName, -1))
			r.URL, _ = url.Parse(strings.Replace(r.URL.String(), accounts.GitlabOrgName, GitlabOrgTestName, -1))
			r.URL, _ = url.Parse(strings.Replace(r.URL.String(), accounts.GitlabUserName, GitlabUserTestName, -1))
		}

		return r.Method == i.Method && (r.URL.String() == i.URL || strings.Contains(i.URL, r.URL.RawPath))
	})

	r.AddSaveFilter(func(i *cassette.Interaction) error {
		if accounts.GithubOrgName != GithubOrgTestName {
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GithubOrgName, GithubOrgTestName, -1)
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GithubUserName, GithubUserTestName, -1)
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GitlabOrgName, GitlabOrgTestName, -1)
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GitlabUserName, GitlabUserTestName, -1)

			i.Request.URL = strings.Replace(i.Request.URL, accounts.GithubOrgName, GithubOrgTestName, -1)
			i.Request.URL = strings.Replace(i.Request.URL, accounts.GithubUserName, GithubUserTestName, -1)
			i.Request.URL = strings.Replace(i.Request.URL, accounts.GitlabOrgName, GitlabOrgTestName, -1)
			i.Request.URL = strings.Replace(i.Request.URL, accounts.GitlabUserName, GitlabUserTestName, -1)

			i.Response.Body = strings.Replace(i.Response.Body, accounts.GithubOrgName, GithubOrgTestName, -1)
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GithubUserName, GithubUserTestName, -1)
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GitlabOrgName, GitlabOrgTestName, -1)
			i.Response.Body = strings.Replace(i.Response.Body, accounts.GitlabUserName, GitlabUserTestName, -1)

			for headerKey, h := range i.Response.Headers {
				for ind, header := range h {
					header = strings.Replace(header, accounts.GithubOrgName, GithubOrgTestName, -1)
					header = strings.Replace(header, accounts.GithubUserName, GithubUserTestName, -1)
					header = strings.Replace(header, accounts.GitlabOrgName, GitlabOrgTestName, -1)
					header = strings.Replace(header, accounts.GitlabUserName, GitlabUserTestName, -1)
					if ind == 0 {
						i.Response.Headers.Set(headerKey, header)
					} else {
						i.Response.Headers.Add(headerKey, header)
					}
				}
			}

		}
		return nil
	})

	return r, nil
}

func getAccounts() *accounts {
	accounts := &accounts{}

	ghOrgName := os.Getenv("GITHUB_ORG_NAME")
	if ghOrgName == "" {
		accounts.GithubOrgName = GithubOrgTestName
	} else {
		accounts.GithubOrgName = ghOrgName
	}

	ghUserName := os.Getenv("GITHUB_USER_NAME")
	if ghUserName == "" {
		accounts.GithubUserName = GithubUserTestName
	} else {
		accounts.GithubUserName = ghUserName
	}

	glOrgName := os.Getenv("GITLAB_ORG_NAME")
	if glOrgName == "" {
		accounts.GitlabOrgName = GitlabOrgTestName
	} else {
		accounts.GitlabOrgName = glOrgName
	}

	glUserName := os.Getenv("GITLAB_USER_NAME")
	if glUserName == "" {
		accounts.GitlabUserName = GitlabUserTestName
	} else {
		accounts.GitlabUserName = glUserName
	}

	return accounts
}

func TestMain(m *testing.M) {
	accounts := getAccounts()

	var err error
	cacheGithubRecorder, err = NewRecorder("github", accounts)
	if err != nil {
		panic(err)
	}

	cacheGitlabRecorder, err = NewRecorder("gitlab", accounts)
	if err != nil {
		panic(err)
	}

	rand.Seed(time.Now().UnixNano())

	exitCode := m.Run()

	err = cacheGithubRecorder.Stop()
	if err != nil {
		panic(err)
	}

	err = cacheGitlabRecorder.Stop()
	if err != nil {
		panic(err)
	}

	os.Exit(exitCode)
}

func newGithubTestClient(customTransportFactory gitprovider.ChainableRoundTripperFunc) (gitprovider.Client, error) {
	token := os.Getenv("GITHUB_TOKEN")
	if token == "" { // This is the case when the tests run in the ci/cd tool. No need to have a value as everything is cached
		token = " "
	}

	return github.NewClient(
		github.WithOAuth2Token(token),
		github.WithPreChainTransportHook(customTransportFactory),
		github.WithDestructiveAPICalls(true),
	)
}

func newGitlabTestClient(customTransportFactory gitprovider.ChainableRoundTripperFunc) (gitprovider.Client, error) {
	token := os.Getenv("GITLAB_TOKEN")
	if token == "" { // This is the case when the tests run in the ci/cd tool. No need to have a value as everything is cached
		token = " "
	}

	return gitlab.NewClient(
		"",
		"",
		gitlab.WithOAuth2Token(token),
		gitlab.WithPreChainTransportHook(customTransportFactory),
		gitlab.WithDestructiveAPICalls(true),
	)
}

func Test_CreatePullRequestToOrgRepo(t *testing.T) {
	accounts := getAccounts()

	githubTestClient, err := newGithubTestClient(SetRecorder(cacheGithubRecorder))
	assert.NoError(t, err)

	gitlabTestClient, err := newGitlabTestClient(SetRecorder(cacheGitlabRecorder))
	assert.NoError(t, err)

	providers := []struct {
		provider string
		client   gitprovider.Client
		domain   string
		orgName  string
		userName string
	}{
		{"github", githubTestClient, GITHUB_DOMAIN, accounts.GithubOrgName, accounts.GithubUserName},
		{"gitlab", gitlabTestClient, GITLAB_DOMAIN, accounts.GitlabOrgName, accounts.GitlabUserName},
	}

	testNameFormat := "create pr for %s account [%s]"
	for _, p := range providers {
		testName := fmt.Sprintf(testNameFormat, "org", p.provider)
		t.Run(testName, func(t *testing.T) {
			CreateTestPullRequestToOrgRepo(t, p.client, p.domain, p.orgName)
		})
		testName = fmt.Sprintf(testNameFormat, "user", p.provider)
		t.Run(testName, func(t *testing.T) {
			CreateTestPullRequestToUserRepo(t, p.client, p.domain, p.userName)
		})

	}
}

func TestCreateRepository(t *testing.T) {
	accounts := getAccounts()

	repoName := "test-org-repo"

	githubTestClient, err := newGithubTestClient(SetRecorder(cacheGithubRecorder))
	assert.NoError(t, err)

	SetGithubProvider(githubTestClient)
	defer SetGithubProvider(nil)

	err = CreateRepository(repoName, accounts.GithubOrgName, true)
	assert.NoError(t, err)
}

func CreateTestPullRequestToOrgRepo(t *testing.T, client gitprovider.Client, domain string, orgName string) {
	repoName := "test-org-repo"
	branchName := "test-org-branch"

	doesNotExistOrg := "doesnotexists"

	orgRepoRef := NewOrgRepositoryRef(domain, orgName, repoName)
	doesNotExistOrgRepoRef := NewOrgRepositoryRef(domain, doesNotExistOrg, repoName)
	repoInfo := NewRepositoryInfo("test org repository", gitprovider.RepositoryVisibilityPrivate)
	opts := &gitprovider.RepositoryCreateOptions{
		AutoInit: gitprovider.BoolVar(true),
	}

	err := CreateOrgRepository(client, orgRepoRef, repoInfo, opts)
	assert.NoError(t, err)

	err = CreateOrgRepository(client, doesNotExistOrgRepoRef, repoInfo, opts)
	assert.Error(t, err)

	path := "setup/config.yaml"
	content := "init content"
	files := []gitprovider.CommitFile{
		{
			Path:    &path,
			Content: &content,
		},
	}

	commitMessage := "added config files"
	prTitle := "config files"
	prDescription := "test description"

	err = CreatePullRequestToOrgRepo(client, orgRepoRef, "", branchName, files, commitMessage, prTitle, prDescription)
	assert.NoError(t, err)

	err = CreatePullRequestToOrgRepo(client, orgRepoRef, "branchdoesnotexists", branchName, files, commitMessage, prTitle, prDescription)
	assert.Error(t, err)

	err = CreatePullRequestToOrgRepo(client, doesNotExistOrgRepoRef, "", branchName, files, commitMessage, prTitle, prDescription)
	assert.Error(t, err)

	t.Cleanup(func() {
		ctx := context.Background()
		org, err := client.OrgRepositories().Get(ctx, orgRepoRef)
		assert.NoError(t, err)
		err = org.Delete(ctx)
		assert.NoError(t, err)
	})
}

func CreateTestPullRequestToUserRepo(t *testing.T, client gitprovider.Client, domain string, userAccount string) {
	repoName := "test-user-repo"
	branchName := "test-user-branch"

	doesnotExistUserAccount := "doesnotexists"

	userRepoRef := NewUserRepositoryRef(domain, userAccount, repoName)
	doesNotExistsUserRepoRef := NewUserRepositoryRef(domain, doesnotExistUserAccount, repoName)
	repoInfo := NewRepositoryInfo("test user repository", gitprovider.RepositoryVisibilityPrivate)
	opts := &gitprovider.RepositoryCreateOptions{
		AutoInit: gitprovider.BoolVar(true),
	}

	err := CreateUserRepository(client, userRepoRef, repoInfo, opts)
	assert.NoError(t, err)

	err = CreateUserRepository(client, doesNotExistsUserRepoRef, repoInfo, opts)
	assert.Error(t, err)

	path := "setup/config.yaml"
	content := "init content"
	files := []gitprovider.CommitFile{
		{
			Path:    &path,
			Content: &content,
		},
	}

	commitMessage := "added config files"
	prTitle := "config files"
	prDescription := "test description"

	err = CreatePullRequestToUserRepo(client, userRepoRef, "", branchName, files, commitMessage, prTitle, prDescription)
	assert.NoError(t, err)

	err = CreatePullRequestToUserRepo(client, userRepoRef, "branchdoesnotexists", branchName, files, commitMessage, prTitle, prDescription)
	assert.Error(t, err)

	err = CreatePullRequestToUserRepo(client, doesNotExistsUserRepoRef, "", branchName, files, commitMessage, prTitle, prDescription)
	assert.Error(t, err)

	t.Cleanup(func() {
		ctx := context.Background()
		user, err := client.UserRepositories().Get(ctx, userRepoRef)
		assert.NoError(t, err)
		err = user.Delete(ctx)
		assert.NoError(t, err)
	})
}

func TestGetOwnerType(t *testing.T) {
	accounts := getAccounts()

	githubTestClient, err := newGithubTestClient(SetRecorder(cacheGithubRecorder))
	assert.NoError(t, err)

	ownerType, err := getOwnerType(githubTestClient, accounts.GithubOrgName)

	assert.NoError(t, err)
	assert.Equal(t, OwnerTypeOrg, ownerType)
}
