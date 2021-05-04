package main

import (
	"context"
	"log"

	"github.com/janakerman/tekton-playground/mono-interceptor/internal/interceptor"
)

func main() {
	err := interceptor.Serve(context.Background())
	if err != nil {
		log.Fatal(err)
	}
}
