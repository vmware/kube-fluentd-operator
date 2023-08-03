package main

import (
	"context"
	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/testcontainers/testcontainers-go"

	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/wait"
)

var mu = sync.Mutex{}
var counterOutput int
var counterTotal = 5

func TestFluentd(t *testing.T) {
	assert := assert.New(t)
	path, err := os.Getwd()
	if err != nil {
		log.Println(err)
	}
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image: "vmware/base-fluentd-operator:latest",
		Env: map[string]string{
			"FLUENTD_OPT": "--no-supervisor",
		},
		SkipReaper: true,
		Mounts: testcontainers.ContainerMounts{
			testcontainers.ContainerMount{
				Source: testcontainers.GenericBindMountSource{
					HostPath: fmt.Sprintf("%s/test", path),
				},
				Target: "/workspace/test",
			},
			testcontainers.ContainerMount{
				Source: testcontainers.GenericBindMountSource{
					HostPath: fmt.Sprintf("%s/test", path),
				},
				Target: "/var/log",
			},
			testcontainers.ContainerMount{
				Source: testcontainers.GenericBindMountSource{
					HostPath: fmt.Sprintf("%s/test/ci.conf", path),
				},
				Target: "/fluentd/etc/fluent.conf",
			},
			testcontainers.ContainerMount{
				Source: testcontainers.GenericBindMountSource{
					HostPath: fmt.Sprintf("%s/test/input.conf", path),
				},
				Target: "/fluentd/etc/input.conf",
			},
		},
		NetworkMode: "host",
		WaitingFor:  wait.ForLog("Found configuration file: /fluentd/etc/fluent.conf"),
	}
	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		log.Println(err)
	}
	defer func() {
		if err := container.Terminate(ctx); err != nil {
			t.Fatalf("failed to terminate container: %s", err.Error())
		}
	}()
	startReceiverServer()
	time.Sleep(15 * time.Second)
	mu.Lock()
	assert.Equal(counterTotal, counterOutput)
	mu.Unlock()
}

func startReceiverServer() {
	server := &http.Server{
		Addr: ":9090",
	}
	http.HandleFunc("/", printLogs)
	go server.ListenAndServe()
}

func printLogs(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	tagName := r.Header["Tag"][0]
	bodyString := string(bodyBytes)
	log.Println("Received data for tag: " + tagName)
	b, err := os.ReadFile("test/results/" + tagName + ".out") // just pass the file name
	if err != nil {
		log.Fatal(err)
	}
	str := string(b) // convert content to a 'string'
	if str != bodyString {
		log.Fatal("Unmatch for tag " + tagName)
	} else {
		mu.Lock()
		counterOutput++
		mu.Unlock()
		log.Println("Matching results for tag " + tagName + " and counter value: " + fmt.Sprint(counterOutput))
	}
}
