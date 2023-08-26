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
	log.Println(path)
	if err != nil {
		log.Println(err)
	}
	ctx := context.Background()

	imageName := os.Getenv("TEST_IMAGE_NAME")
	if imageName == "" {
		imageName = "vmware/base-fluentd-operator"
	}
	imageTag := os.Getenv("TEST_IMAGE_TAG")
	if imageTag != "" {
		imageName = fmt.Sprintf("%s:%s", imageName, imageTag)
	}

	req := testcontainers.ContainerRequest{
		Image: imageName,
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
	time.Sleep(30 * time.Second)
	mu.Lock()
	assert.Equal(counterTotal, counterOutput)
	mu.Unlock()
}

func startReceiverServer() {
	server := &http.Server{
		Addr: "0.0.0.0:9090",
	}
	http.HandleFunc("/", printLogs)
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not start server: %s", err.Error())
		}
	}()

}

func printLogs(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error reading request body: %v", err)
		return
	}

	tags, ok := r.Header["Tag"]
	if !ok || len(tags) == 0 {
		http.Error(w, "Tag header not found", http.StatusBadRequest)
	}
	tagName := tags[0]

	bodyString := string(bodyBytes)
	log.Printf("Received data for tag: %s", tagName)

	b, err := os.ReadFile(fmt.Sprintf("test/results/%s.out", tagName))
	if err != nil {
		log.Printf("Error reading result file for tag %s: %v", tagName, err)
	}

	str := string(b)
	log.Printf("Result from file: %s", str)

	if str != bodyString {
		log.Printf("Unmatch for tag %s", tagName)
		http.Error(w, "Mismatched data", http.StatusBadRequest)
		return
	}

	mu.Lock()
	defer mu.Unlock() // Always release the lock
	counterOutput++
	log.Printf("Matching results for tag %s and counter value: %d", tagName, counterOutput)
}
