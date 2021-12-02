package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"time"

	"github.com/benbjohnson/clock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type testServerTransport struct {
	testServeUrl string
	roundTripper http.RoundTripper
}

func (t *testServerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Fake out the client but preserve the URL, as the URLs are key to validating that
	// the authHandler is working.
	tsUrl, err := url.Parse(t.testServeUrl)
	if err != nil {
		return nil, err
	}

	tsUrl.Path = r.URL.Path

	r.URL = tsUrl

	return t.roundTripper.RoundTrip(r)
}

var _ = Describe("Github Device Flow", func() {
	var ts *httptest.Server
	var client *http.Client
	token := "gho_sUpErSecRetToKeN"
	userCode := "ABC-123"
	verificationUri := "http://somegithuburl.com"

	var _ = BeforeEach(func() {
		ts = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			// Quick and dirty router to simulate the Github API
			if strings.Contains(r.URL.Path, "/device/code") {
				err := json.NewEncoder(w).Encode(&GithubDeviceCodeResponse{
					DeviceCode:      "123456789",
					UserCode:        userCode,
					VerificationURI: verificationUri,
					Interval:        1,
				})
				Expect(err).NotTo(HaveOccurred())

			}

			if strings.Contains(r.URL.Path, "/oauth/access_token") {
				err := json.NewEncoder(w).Encode(&githubAuthResponse{
					AccessToken: token,
					Error:       "",
				})
				Expect(err).NotTo(HaveOccurred())
			}
		}))

		client = ts.Client()
		client.Transport = &testServerTransport{testServeUrl: ts.URL, roundTripper: client.Transport}
	})

	var _ = AfterEach(func() {
		ts.Close()
	})

	It("does the auth flow", func() {
		authHandler := NewGithubDeviceFlowHandler(client)

		var cliOutput bytes.Buffer
		result, err := authHandler(context.Background(), &cliOutput)

		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(token))
		// We need to ensure the user code and verification url are in the CLI ouput.
		// Check for the prescense of substrings to avoid failing tests on trivial output changes.
		Expect(cliOutput.String()).To(ContainSubstring(userCode))
		Expect(cliOutput.String()).To(ContainSubstring(verificationUri))
	})
	Describe("pollAuthStatus", func() {
		XIt("retries after a slow_down response from github", func() {
			rt := newMockRoundTripper(3, token)
			client.Transport = &testServerTransport{testServeUrl: ts.URL, roundTripper: rt}
			interval := 5 * time.Second

			c := clock.NewMock()

			go func() {
				_, err := pollAuthStatus(c.Sleep, interval, client, "somedevicecode")
				Expect(err).NotTo(HaveOccurred())
			}()
			runtime.Gosched()

			// +5
			c.Add(5 * time.Second)
			Expect(rt.calls).To(Equal(1), "should have tried the first time")

			// +10
			c.Add(5 * time.Second)
			Expect(rt.calls).To(Equal(1), "should NOT have retried early")

			// +20
			c.Add(10 * time.Second)
			Expect(rt.calls).To(Equal(2), "should have backed off 10 seconds")
		})
		It("returns a token after a slow_down", func() {
			rt := newMockRoundTripper(1, token)
			client.Transport = &testServerTransport{testServeUrl: ts.URL, roundTripper: rt}
			interval := 5 * time.Second
			c := clock.NewMock()

			var resultToken string
			var err error
			go func() {
				resultToken, err = pollAuthStatus(c.Sleep, interval, client, "somedevicecode")
				Expect(err).NotTo(HaveOccurred())
			}()
			runtime.Gosched()

			c.Add(5 * time.Second)
			Expect(rt.calls).To(Equal(1), "should have tried the first time")

			c.Add(15 * time.Second)
			Expect(rt.calls).To(Equal(2), "should have added 10 seconds of back off")

			Expect(resultToken).To(Equal(token))
		})
	})
})

type mockAuthRoundTripper struct {
	fn    func(r *http.Request) (*http.Response, error)
	calls int
}

func (rt *mockAuthRoundTripper) MockRoundTrip(fn func(r *http.Request) (*http.Response, error)) {
	rt.fn = fn
}

func (rt *mockAuthRoundTripper) RoundTrip(r *http.Request) (*http.Response, error) {
	return rt.fn(r)
}

func newMockRoundTripper(pollCount int, token string) *mockAuthRoundTripper {
	rt := &mockAuthRoundTripper{calls: 0}

	rt.MockRoundTrip(func(r *http.Request) (*http.Response, error) {
		b := bytes.NewBuffer(nil)

		data := githubAuthResponse{Error: "slow_down"}
		if rt.calls == pollCount {
			data = githubAuthResponse{Error: "", AccessToken: token}
		}

		if err := json.NewEncoder(b).Encode(data); err != nil {
			return nil, err
		}

		res := &http.Response{
			Body: io.NopCloser(b),
		}

		res.StatusCode = http.StatusOK

		rt.calls = rt.calls + 1
		return res, nil
	})

	return rt
}
