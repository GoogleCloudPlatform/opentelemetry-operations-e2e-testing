// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/pubsub/pstest"
	"cloud.google.com/go/storage"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type fakeStore struct {
	objs    []*storage.ObjectAttrs
	deleted []string
}

func (f *fakeStore) List(context.Context) ([]*storage.ObjectAttrs, error) { return f.objs, nil }
func (f *fakeStore) DeleteLock(_ context.Context, obj string, _ int64) error {
	f.deleted = append(f.deleted, obj)
	return nil
}

func makeObj(name string, updated time.Time, size, gen int64) *storage.ObjectAttrs {
	return &storage.ObjectAttrs{Name: name, Updated: updated, Size: size, Generation: gen}
}

func TestParseFlags(t *testing.T) {
	bad := [][]string{
		{},
		{"--project-id=p", "--execute"},
		{"--project-id=p", "extra-arg", "--limit=1"},
		{"--project-id=p", "--min-age=-1h"},
		{"--project-id=p", "--min-age=0s"},
		{"--project-id=p", "--limit=-1"},
		{"--project-id=p", "--pubsub-topic="},
		{"--project-id=p", "--pubsub-topic=projects/p/topics/e2e-cleanup"},
		{"--project-id=p", "--pubsub-topic=bad topic"},
	}
	for _, args := range bad {
		if _, err := parseFlags(args); err == nil {
			t.Errorf("expected error for args %v", args)
		}
	}

	cfg, err := parseFlags([]string{"--project-id=p", "--image=img:1", "--execute"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.pubsubTopic != defaultPubSubTopic || !cfg.execute {
		t.Errorf("unexpected parsed config: %+v", cfg)
	}
}

func TestMarshalCleanupPubSubMessage(t *testing.T) {
	raw, err := marshalCleanupPubSubMessage("0e30aee22fe1", "us-central1-docker.pkg.dev/p/r/img:latest")
	if err != nil {
		t.Fatal(err)
	}
	var decoded cleanupPubSubMessage
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != "0e30aee22fe1" {
		t.Errorf("id = %q, want 0e30aee22fe1", decoded.ID)
	}
	if decoded.Status != "FAILURE" {
		t.Errorf("status = %q, want FAILURE", decoded.Status)
	}
	if got := decoded.Substitutions["_TEST_RUNNER_IMAGE"]; got != "us-central1-docker.pkg.dev/p/r/img:latest" {
		t.Errorf("_TEST_RUNNER_IMAGE = %q", got)
	}
}

func TestSelectTargets(t *testing.T) {
	cutoff := time.Date(2026, 11, 13, 0, 0, 0, 0, time.UTC)
	oldest := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	old := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	recent := time.Date(2026, 11, 19, 12, 0, 0, 0, time.UTC)

	const (
		hexLocked  = "0e30aee22fe1"
		uuidOpen   = "10cbaa82-039b-4fb1-9f45-d8b5284f041b"
		hexRecent  = "abcdef012345"
		hexNewLock = "fedcba987654"
	)

	objs := []*storage.ObjectAttrs{
		makeObj("terraform/state/default.tfstate", oldest, 126625, 1),
		makeObj("terraform/state/persistent-collector.tfstate", oldest, 50000, 10),
		makeObj("terraform/state/"+hexLocked+".tfstate", old, 17000, 3),
		makeObj("terraform/state/"+hexLocked+".tflock", old, 284, 33),
		makeObj("terraform/state/"+uuidOpen+".tfstate", oldest, 20000, 2),
		makeObj("terraform/state/"+hexRecent+".tfstate", recent, 15000, 4),
		makeObj("terraform/state/"+hexNewLock+".tfstate", old, 15000, 5),
		makeObj("terraform/state/"+hexNewLock+".tflock", recent, 284, 55),
		makeObj("terraform/state/nested/"+uuidOpen+".tfstate", oldest, 999, 6),
	}

	targets, skipped := selectTargets(objs, cutoff, 0)
	if skipped != 2 || len(targets) != 2 {
		t.Fatalf("got %d targets, %d skipped; want 2, 2", len(targets), skipped)
	}
	if targets[0].Name != uuidOpen || targets[0].LockObject != "" {
		t.Errorf("expected oldest workspace %s first, got %+v", uuidOpen, targets[0])
	}
	if targets[1].Name != hexLocked || targets[1].LockGeneration != 33 {
		t.Errorf("unexpected targets[1]: %+v", targets[1])
	}

	limited, _ := selectTargets(objs, cutoff, 1)
	if len(limited) != 1 || limited[0].Name != uuidOpen {
		t.Errorf("expected limit=1 -> [%s], got %+v", uuidOpen, limited)
	}
}

func TestDryRunAndExecuteOperationLogs(t *testing.T) {
	old := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	objs := []*storage.ObjectAttrs{
		makeObj("terraform/state/default.tfstate", old, 1000, 1),
		makeObj("terraform/state/0e30aee22fe1.tfstate", old, 100, 2),
		makeObj("terraform/state/0e30aee22fe1.tflock", old, 50, 22),
	}

	t.Run("dry-run", func(t *testing.T) {
		cfg, err := parseFlags([]string{"--project-id=test-proj"})
		if err != nil {
			t.Fatal(err)
		}
		store := &fakeStore{objs: objs}
		publish := func(context.Context, *slog.Logger, string, string) error {
			t.Fatal("publisher must not be called in dry-run")
			return nil
		}
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		if err := run(context.Background(), cfg, store, publish, logger); err != nil {
			t.Fatal(err)
		}
		if len(store.deleted) != 0 {
			t.Fatalf("unexpected lock deletions in dry-run: %v", store.deleted)
		}
		out := buf.String()
		for _, want := range []string{
			`msg="DRY-RUN: delete stale lock"`,
			`workspace=0e30aee22fe1`,
			`object=gs://test-proj-e2e-tfstate/terraform/state/0e30aee22fe1.tflock`,
			`generation=22`,
			`msg="DRY-RUN: publish cleanup event"`,
			`project=test-proj`,
			`topic=e2e-cleanup`,
			`state_size=100`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("expected log substring %q in output:\n%s", want, out)
			}
		}
	})

	t.Run("execute", func(t *testing.T) {
		cfg, err := parseFlags([]string{"--project-id=test-proj", "--image=gcr.io/p/img:1", "--execute"})
		if err != nil {
			t.Fatal(err)
		}
		store := &fakeStore{objs: objs}
		var published []string
		publish := func(_ context.Context, wlog *slog.Logger, image, ws string) error {
			if image != "gcr.io/p/img:1" {
				t.Errorf("unexpected image: %q", image)
			}
			wlog.Info("cleanup event published", "topic", "e2e-cleanup", "message_id", "m-1")
			published = append(published, ws)
			return nil
		}
		var buf bytes.Buffer
		logger := slog.New(slog.NewTextHandler(&buf, nil))
		if err := run(context.Background(), cfg, store, publish, logger); err != nil {
			t.Fatal(err)
		}
		if len(store.deleted) != 1 || store.deleted[0] != "terraform/state/0e30aee22fe1.tflock" {
			t.Fatalf("expected lock deletion, got %v", store.deleted)
		}
		if len(published) != 1 || published[0] != "0e30aee22fe1" {
			t.Fatalf("expected publish for 0e30aee22fe1, got %v", published)
		}
		out := buf.String()
		if strings.Contains(out, "DRY-RUN:") {
			t.Errorf("unexpected DRY-RUN prefix in execute output:\n%s", out)
		}
		for _, want := range []string{
			`msg="delete stale lock"`,
			`msg="publish cleanup event"`,
			`topic=e2e-cleanup`,
			`msg="cleanup event published"`,
			`message_id=m-1`,
			`msg="workspace cleanup enqueued"`,
			`progress=1/1`,
		} {
			if !strings.Contains(out, want) {
				t.Errorf("expected log substring %q in output:\n%s", want, out)
			}
		}
	})
}

func TestNewPubSubPublisher(t *testing.T) {
	ctx := context.Background()
	srv := pstest.NewServer()
	t.Cleanup(func() { _ = srv.Close() })

	conn, err := grpc.NewClient(srv.Addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	client, err := pubsub.NewClient(ctx, "test-proj", option.WithGRPCConn(conn))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })

	topic, err := client.CreateTopic(ctx, "e2e-cleanup")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(topic.Stop)

	publish := newPubSubPublisher(topic)
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	if err := publish(ctx, logger, "gcr.io/p/runner:latest", "0e30aee22fe1"); err != nil {
		t.Fatal(err)
	}

	msgs := srv.Messages()
	if len(msgs) != 1 {
		t.Fatalf("got %d published messages, want 1", len(msgs))
	}
	var got cleanupPubSubMessage
	if err := json.Unmarshal(msgs[0].Data, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != "0e30aee22fe1" || got.Status != "FAILURE" || got.Substitutions["_TEST_RUNNER_IMAGE"] != "gcr.io/p/runner:latest" {
		t.Errorf("unexpected published payload: %+v", got)
	}
}
