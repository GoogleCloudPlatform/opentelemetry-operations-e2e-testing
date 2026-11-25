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

// Command cleanupworkspaces discovers stale Terraform workspaces in the e2e
// GCS state bucket, clears stale locks, and publishes cleanup events to the
// e2e-cleanup Pub/Sub topic. Dry-run by default; pass --execute to run.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"cloud.google.com/go/pubsub"
	"cloud.google.com/go/storage"
	"golang.org/x/sync/errgroup"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

const (
	defaultWorkspace   = "default"
	defaultPubSubTopic = "e2e-cleanup"
	statePrefix        = "terraform/state/"
	maxConcurrency     = 10
)

// ephemeralWorkspaceRe matches only ephemeral e2e test run IDs: legacy 12-char
// hex IDs or Cloud Build UUIDs. Named workspaces (e.g. "default") never match.
var ephemeralWorkspaceRe = regexp.MustCompile(`^([0-9a-f]{12}|[0-9a-f-]{36})$`)

type config struct {
	projectID   string
	image       string
	pubsubTopic string
	minAge      time.Duration
	limit       int
	execute     bool
}

type workspaceTarget struct {
	Name           string
	StateUpdated   time.Time
	StateSize      int64
	LockObject     string
	LockGeneration int64
	LockUpdated    time.Time
}

type stateStore interface {
	List(ctx context.Context) ([]*storage.ObjectAttrs, error)
	DeleteLock(ctx context.Context, object string, generation int64) error
}

type gcsStore struct {
	bucket *storage.BucketHandle
}

func (s *gcsStore) List(ctx context.Context) ([]*storage.ObjectAttrs, error) {
	var out []*storage.ObjectAttrs
	it := s.bucket.Objects(ctx, &storage.Query{Prefix: statePrefix, Delimiter: "/"})
	for {
		attrs, err := it.Next()
		if errors.Is(err, iterator.Done) {
			return out, nil
		}
		if err != nil {
			return nil, err
		}
		if attrs.Name != "" {
			out = append(out, attrs)
		}
	}
}

func (s *gcsStore) DeleteLock(ctx context.Context, object string, generation int64) error {
	if generation <= 0 {
		return fmt.Errorf("refusing unconditional lock delete for %s: invalid generation %d", object, generation)
	}
	return s.bucket.Object(object).If(storage.Conditions{GenerationMatch: generation}).Delete(ctx)
}

type cleanupPublisher func(ctx context.Context, logger *slog.Logger, image, workspace string) error

// cleanupPubSubMessage matches the Cloud Build notification payload format
// consumed by the global-e2e-cleanup Pub/Sub trigger in tf/persistent.
type cleanupPubSubMessage struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	Substitutions map[string]string `json:"substitutions"`
}

func marshalCleanupPubSubMessage(workspace, image string) ([]byte, error) {
	return json.Marshal(cleanupPubSubMessage{
		ID:     workspace,
		Status: "FAILURE",
		Substitutions: map[string]string{
			"_TEST_RUNNER_IMAGE": image,
		},
	})
}

func newPubSubPublisher(topic *pubsub.Topic) cleanupPublisher {
	return func(ctx context.Context, logger *slog.Logger, image, workspace string) error {
		payload, err := marshalCleanupPubSubMessage(workspace, image)
		if err != nil {
			return fmt.Errorf("marshaling pubsub payload: %w", err)
		}
		msgID, err := topic.Publish(ctx, &pubsub.Message{Data: payload}).Get(ctx)
		if err != nil {
			return fmt.Errorf("publishing to %s: %w", topic.ID(), err)
		}
		logger.Info("cleanup event published", "topic", topic.ID(), "message_id", msgID)
		return nil
	}
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stderr, nil))
	cfg, err := parseFlags(os.Args[1:])
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(0)
		}
		logger.Error("invalid flags", "err", err)
		os.Exit(1)
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	clientOpt := option.WithQuotaProject(cfg.projectID)
	storageClient, err := storage.NewClient(ctx, clientOpt)
	if err != nil {
		logger.Error("creating storage client", "err", err)
		os.Exit(1)
	}
	defer storageClient.Close()

	var publish cleanupPublisher
	if cfg.execute {
		psClient, err := pubsub.NewClient(ctx, cfg.projectID, clientOpt)
		if err != nil {
			logger.Error("creating pubsub client", "err", err)
			os.Exit(1)
		}
		defer psClient.Close()
		topic := psClient.Topic(cfg.pubsubTopic)
		defer topic.Stop()
		publish = newPubSubPublisher(topic)
	}

	store := &gcsStore{bucket: storageClient.Bucket(cfg.projectID + "-e2e-tfstate")}
	if err := run(ctx, cfg, store, publish, logger); err != nil {
		logger.Error("cleanup failed", "err", err)
		os.Exit(1)
	}
}

func parseFlags(args []string) (config, error) {
	fs := flag.NewFlagSet("cleanupworkspaces", flag.ContinueOnError)
	var cfg config
	fs.StringVar(&cfg.projectID, "project-id", "", "GCP project ID (required)")
	fs.StringVar(&cfg.image, "image", "", "Test runner container image (required with --execute)")
	fs.StringVar(&cfg.pubsubTopic, "pubsub-topic", defaultPubSubTopic, "Pub/Sub topic ID in --project-id to publish cleanup events to")
	fs.DurationVar(&cfg.minAge, "min-age", 7*24*time.Hour, "Minimum age of state/lock objects before cleanup")
	fs.IntVar(&cfg.limit, "limit", 0, "Maximum number of workspaces to process (0 for no limit)")
	fs.BoolVar(&cfg.execute, "execute", false, "Clear stale locks and publish cleanup events (default is dry-run)")
	if err := fs.Parse(args); err != nil {
		return cfg, err
	}
	if fs.NArg() > 0 {
		return cfg, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if cfg.projectID = strings.TrimSpace(cfg.projectID); cfg.projectID == "" || strings.ContainsAny(cfg.projectID, " \t\r\n") {
		return cfg, errors.New("--project-id is required")
	}
	if cfg.image = strings.TrimSpace(cfg.image); cfg.execute && (cfg.image == "" || strings.ContainsAny(cfg.image, " \t\r\n")) {
		return cfg, errors.New("--image is required when --execute is set")
	}
	if cfg.pubsubTopic = strings.TrimSpace(cfg.pubsubTopic); cfg.pubsubTopic == "" || strings.ContainsAny(cfg.pubsubTopic, " \t\r\n/") {
		return cfg, fmt.Errorf("--pubsub-topic must be a non-empty bare topic ID without '/' or whitespace, got %q", cfg.pubsubTopic)
	}
	if cfg.minAge <= 0 {
		return cfg, fmt.Errorf("--min-age must be > 0, got %s", cfg.minAge)
	}
	if cfg.limit < 0 {
		return cfg, fmt.Errorf("--limit must be >= 0, got %d", cfg.limit)
	}
	return cfg, nil
}

func run(ctx context.Context, cfg config, store stateStore, publish cleanupPublisher, logger *slog.Logger) error {
	objects, err := store.List(ctx)
	if err != nil {
		return fmt.Errorf("listing state objects: %w", err)
	}

	targets, skipped := selectTargets(objects, time.Now().UTC().Add(-cfg.minAge), cfg.limit)
	logger.Info("selected stale workspaces",
		"count", len(targets),
		"min_age", cfg.minAge,
		"skipped_recent", skipped,
		"excluded", defaultWorkspace,
		"dry_run", !cfg.execute,
	)

	if cfg.execute {
		return executeCleanup(ctx, cfg, store, publish, logger, targets)
	}
	for _, t := range targets {
		if err := cleanupOne(ctx, cfg, store, publish, logger, t); err != nil {
			return err
		}
	}
	return nil
}

func executeCleanup(ctx context.Context, cfg config, store stateStore, publish cleanupPublisher, logger *slog.Logger, targets []workspaceTarget) error {
	if len(targets) == 0 {
		return nil
	}

	var (
		g      errgroup.Group
		done   atomic.Int64
		failed atomic.Int64
	)
	g.SetLimit(maxConcurrency)
	total := int64(len(targets))

	for _, t := range targets {
		t := t
		g.Go(func() error {
			start := time.Now()
			err := cleanupOne(ctx, cfg, store, publish, logger, t)
			n := done.Add(1)
			progress := fmt.Sprintf("%d/%d", n, total)
			elapsed := time.Since(start).Round(time.Second)
			if err != nil {
				failed.Add(1)
				logger.Error("workspace cleanup failed", "progress", progress, "workspace", t.Name, "duration", elapsed, "err", err)
				return nil
			}
			logger.Info("workspace cleanup enqueued", "progress", progress, "workspace", t.Name, "duration", elapsed)
			return nil
		})
	}
	_ = g.Wait()

	failedCount := failed.Load()
	logger.Info("cleanup complete", "succeeded", total-failedCount, "failed", failedCount, "total", total)
	if failedCount > 0 {
		return fmt.Errorf("%d of %d workspace cleanup(s) failed", failedCount, total)
	}
	return nil
}

func selectTargets(objects []*storage.ObjectAttrs, cutoff time.Time, limit int) ([]workspaceTarget, int) {
	states := map[string]*storage.ObjectAttrs{}
	locks := map[string]*storage.ObjectAttrs{}

	for _, obj := range objects {
		rel, ok := strings.CutPrefix(obj.Name, statePrefix)
		if !ok {
			continue
		}
		if ws, ok := strings.CutSuffix(rel, ".tfstate"); ok && ws != "" {
			states[ws] = obj
		} else if ws, ok := strings.CutSuffix(rel, ".tflock"); ok && ws != "" {
			locks[ws] = obj
		}
	}

	var targets []workspaceTarget
	skipped := 0
	for ws, st := range states {
		if ws == defaultWorkspace || !ephemeralWorkspaceRe.MatchString(ws) {
			continue
		}
		if st.Updated.IsZero() || !st.Updated.Before(cutoff) {
			skipped++
			continue
		}
		t := workspaceTarget{Name: ws, StateUpdated: st.Updated, StateSize: st.Size}
		if lk, ok := locks[ws]; ok {
			if lk.Updated.IsZero() || !lk.Updated.Before(cutoff) {
				skipped++
				continue
			}
			t.LockObject = lk.Name
			t.LockGeneration = lk.Generation
			t.LockUpdated = lk.Updated
		}
		targets = append(targets, t)
	}

	sort.Slice(targets, func(i, j int) bool {
		if !targets[i].StateUpdated.Equal(targets[j].StateUpdated) {
			return targets[i].StateUpdated.Before(targets[j].StateUpdated)
		}
		return targets[i].Name < targets[j].Name
	})
	if limit > 0 && len(targets) > limit {
		targets = targets[:limit]
	}
	return targets, skipped
}

func cleanupOne(ctx context.Context, cfg config, store stateStore, publish cleanupPublisher, logger *slog.Logger, t workspaceTarget) error {
	prefix := ""
	if !cfg.execute {
		prefix = "DRY-RUN: "
	}
	bucket := cfg.projectID + "-e2e-tfstate"
	wlog := logger.With("workspace", t.Name)

	if t.LockObject != "" {
		wlog.Info(prefix+"delete stale lock",
			"object", fmt.Sprintf("gs://%s/%s", bucket, t.LockObject),
			"generation", t.LockGeneration,
			"lock_updated", t.LockUpdated.UTC().Format(time.RFC3339),
		)
		if cfg.execute {
			if err := store.DeleteLock(ctx, t.LockObject, t.LockGeneration); err != nil {
				return fmt.Errorf("unlocking %s#%d: %w", t.LockObject, t.LockGeneration, err)
			}
		}
	}

	wlog.Info(prefix+"publish cleanup event",
		"project", cfg.projectID,
		"topic", cfg.pubsubTopic,
		"state_updated", t.StateUpdated.UTC().Format(time.RFC3339),
		"state_size", t.StateSize,
	)
	if !cfg.execute {
		return nil
	}
	return publish(ctx, wlog, cfg.image, t.Name)
}
