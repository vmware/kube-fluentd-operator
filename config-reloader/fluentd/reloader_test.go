// Copyright © 2018 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: BSD-2-Clause

package fluentd

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNullReloader(t *testing.T) {
	var r *Reloader
	r.ReloadConfiguration()
}
func TestReloaderCalls(t *testing.T) {
	ctx := context.Background()
	port := 11543

	counter := 0

	handler := func(w http.ResponseWriter, r *http.Request) {
		fmt.Printf("req %+v", r)
		if r.Method == "POST" && r.RequestURI == "/api/config.reload" {
			counter++
		}
	}

	server := http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: http.HandlerFunc(handler),
	}

	go server.ListenAndServe()
	defer server.Close()

	r := NewReloader(ctx, port)

	var err error
	for i := 10; i >= 0; i-- {
		_, err = http.Get(fmt.Sprintf("http://127.0.0.1:%d", port))
		if err != nil {
			time.Sleep(200 * time.Millisecond)
		}
	}

	if err != nil {
		t.Fatalf("Mock server not up within reasonable time")
	}

	r.ReloadConfiguration()
	r.ReloadConfiguration()
	r.ReloadConfiguration()

	assert.Equal(t, 3, counter)
}
