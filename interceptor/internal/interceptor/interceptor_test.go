package interceptor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
)

func TestInterceptor(t *testing.T) {
	fullRepoName := "org/repo"
	beforeRef := "beforeRef"
	afterRef := "afterRef"
	changedFiles := []*github.CommitFile{
		{Filename: github.String("folder/subfolder1/file")},
		{PreviousFilename: github.String("folder/subfolder2/file")},
		{Filename: github.String("folder/subfolder3/file")},
	}

	githubServer := githubMock(t, fullRepoName, beforeRef, afterRef, changedFiles)
	require.NoError(t, os.Setenv(EnvGithubHost, githubServer.URL+"/"))

	shutdown := startInterceptor(t)
	t.Cleanup(shutdown)

	event := github.PushEvent{
		Before: &beforeRef,
		After:  &afterRef,
		Repo: &github.PushEventRepository{
			FullName: &fullRepoName,
		},
	}
	githubPrBody, err := json.Marshal(event)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodPost, "http://localhost:8080/", bytes.NewReader(githubPrBody))
	require.NoError(t, err)

	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer res.Body.Close()

	assert.Equal(t, 200, res.StatusCode)

	b, err := io.ReadAll(res.Body)
	require.NoError(t, err)

	var iRes v1alpha1.InterceptorResponse
	err = json.Unmarshal(b, &iRes)
	require.NoError(t, err)

	assert.Equal(t, []interface{}{
		"folder/subfolder1/file",
		"folder/subfolder2/file",
		"folder/subfolder3/file",
	}, iRes.Extensions["filesChanged"])

	assert.Equal(t, codes.OK, iRes.Status.Code)
	assert.Equal(t, "", iRes.Status.Message, "")
	assert.True(t, iRes.Continue)
}

func githubMock(t *testing.T, fullName, base, head string, files []*github.CommitFile) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		expectedURL := fmt.Sprintf("/repos/%s/compare/%s...%s", fullName, base, head)
		require.True(t, strings.HasPrefix(req.URL.Path, expectedURL), "url path '%s' does not match expected '%s'", req.URL.Path, expectedURL)

		res := github.CommitsComparison{
			Files: files,
		}

		b, err := json.Marshal(res)
		require.NoError(t, err)

		_, err = w.Write(b)
		require.NoError(t, err)
	}))
}

func startInterceptor(t *testing.T) context.CancelFunc {
	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		err := Serve(ctx)
		if err != nil {
			t.Fatalf("failed to start interceptor: %v", err)
		}
	}()

	tick := time.Tick(50 * time.Millisecond)
	timeout := time.After(1 * time.Second)

	select {
	case <-tick:
		req, err := http.NewRequest(http.MethodGet, "http://localhost/health", bytes.NewReader(nil))
		if err != nil {
			log.Fatalf("health request: %v\n", err)
		}
		res, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Printf("health check failed: %v\n", err)
		}
		if res.StatusCode != http.StatusOK {
			log.Printf("health check failed (%d): %v\n", res.StatusCode, err)
		}
		log.Printf("health check passed\n")
		break
	case <-timeout:
		t.Fatalf("timeout out waiting for health check to pass\n")
	}

	return cancel
}
