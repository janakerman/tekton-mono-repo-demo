package interceptor

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/go-github/v35/github"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
)

const EnvGithubHost = "GITHUB_URL"

// Begins serving, blocking until context cancelled.
func Serve(ctx context.Context) error {
	httpClient := &http.Client{}
	ghClient := github.NewClient(httpClient)

	githubUrl, err := url.Parse(os.Getenv(EnvGithubHost))
	if err != nil {
		return err
	}
	log.Printf("github host: %s", githubUrl)
	ghClient.BaseURL = githubUrl

	m := http.NewServeMux()
	m.HandleFunc("/health", handleHealth)
	m.HandleFunc("/", interceptorHandler(ghClient))

	srv := &http.Server{
		Addr:    ":8080",
		Handler: m,
	}

	go func() {
		log.Printf("listening on 8080\n")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("stopped listening: %v\n", err)
		}
	}()
	log.Printf("intercepting\n")

	<-ctx.Done()

	log.Printf("shutting down\n")

	ctxShutDown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = srv.Shutdown(ctxShutDown)
	if err != nil {
		if err != http.ErrServerClosed {
			return fmt.Errorf("shutdown failed: %w", err)
		}
	}

	log.Printf("shut down\n")
	return nil
}

func respondErr(w http.ResponseWriter, statusCode int, code codes.Code, err error) {
	log.Printf("respond %d: %s", statusCode, err)

	b, err := json.Marshal(v1alpha1.InterceptorResponse{
		Continue: false,
		Status: v1alpha1.Status{
			Code:    code,
			Message: err.Error(),
		},
	})
	if err != nil {
		log.Printf("marshall error response: %v", err)
	}

	w.WriteHeader(statusCode)
	_, err = w.Write(b)
	if err != nil {
		log.Printf("write error response: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(200)
}

func interceptorHandler(ghClient *github.Client) func(w http.ResponseWriter, req *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		b, err := io.ReadAll(req.Body)
		if err != nil {
			respondErr(w, 400, codes.InvalidArgument, err)
			return
		}
		defer req.Body.Close()

		log.Printf("handling event: %s", string(b))

		var event *github.PushEvent
		err = json.Unmarshal(b, &event)
		if err != nil {
			respondErr(w, 400, codes.InvalidArgument, err)
			return
		}

		log.Printf("event: %+v", event)

		filesChanged, err := filesChanged(req.Context(), ghClient, event.Repo.GetFullName(), event.GetBefore(), event.GetAfter())
		if err != nil {
			respondErr(w, 500, codes.Unknown, err)
			return
		}

		log.Printf("files changed: %+v", filesChanged)

		iRes := &v1alpha1.InterceptorResponse{
			Extensions: map[string]interface{}{
				"filesChanged": filesChanged,
			},
			Continue: true,
			Status: v1alpha1.Status{
				Code: codes.OK,
			},
		}

		b, err = json.Marshal(iRes)
		if err != nil {
			respondErr(w, 500, codes.Unknown, err)
			return
		}

		log.Printf("responded: %s", string(b))

		w.Header().Add("Content-Type", "application/json")
		_, err = w.Write(b)
		if err != nil {
			respondErr(w, 500, codes.Unknown, err)
			return
		}
	}
}

func filesChanged(ctx context.Context, ghClient *github.Client, fullRepoName, fromSha, toSha string) ([]string, error) {
	split := strings.Split(fullRepoName, "/")
	if len(split) != 2 {
		return nil, fmt.Errorf("repo name not in format <owner>/<repo> %s", fullRepoName)
	}

	compare, _, err := ghClient.Repositories.CompareCommits(ctx, split[0], split[1], fromSha, toSha)
	if err != nil {
		return nil, fmt.Errorf("compare %s/%s %s to %s: %w", split[0], split[1], fromSha, toSha, err)
	}

	var changedFiles []string
	for _, f := range compare.Files {
		if f.GetFilename() != "" {
			changedFiles = append(changedFiles, f.GetFilename())
		}
		if f.GetPreviousFilename() != "" {
			changedFiles = append(changedFiles, f.GetPreviousFilename())
		}
	}

	return changedFiles, nil
}
