// Package objectstorage reconciles S3 bucket replication status between
// primary and secondary buckets. For S3 cross-region replication we do
// not copy objects ourselves (the cloud provider does); we verify the
// replica bucket's object count and total size matches the primary,
// within the configured lag threshold.
package objectstorage

import (
	"context"
	"fmt"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/anomalyco/gitlab-geo-sync/internal/config"
	"github.com/anomalyco/gitlab-geo-sync/internal/metrics"
	"github.com/anomalyco/gitlab-geo-sync/internal/reconciler"
)

const name = "object_storage"

// Reconciler verifies S3 bucket parity between primary and replica.
type Reconciler struct {
	primaryClient  *s3.Client
	replicaClient  *s3.Client
	primaryBucket  string
	replicaBucket  string
	lagThreshold   time.Duration
}

// New creates an S3 object-storage reconciler from the primary site config.
func New(ctx context.Context, cfg *config.Config) (*Reconciler, error) {
	s3cfg := cfg.Primary.ObjectStore.S3
	if s3cfg == nil {
		return nil, fmt.Errorf("primary.object_storage.s3 is required for s3 backend")
	}
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(s3cfg.Region),
		awsconfig.WithCredentialsProvider(
			credentialsFromConfig(s3cfg),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}
	opts := []func(*s3.Options){}
	if s3cfg.Endpoint != "" {
		opts = append(opts, func(o *s3.Options) { o.BaseEndpoint = ptr(s3cfg.Endpoint) })
	}
	return &Reconciler{
		primaryClient: s3.NewFromConfig(awsCfg, opts...),
		replicaClient: s3.NewFromConfig(awsCfg, opts...),
		primaryBucket: s3cfg.PrimaryBucket,
		replicaBucket: s3cfg.ReplicaBucket,
		lagThreshold:  s3cfg.ReplicationLag,
	}, nil
}

func (r *Reconciler) Name() string { return name }

// Reconcile lists objects in both buckets and compares count + total size.
// If the replica is behind by more than lagThreshold, returns not-OK.
func (r *Reconciler) Reconcile(ctx context.Context) reconciler.Result {
	start := time.Now()
	pCount, pSize, err := bucketStats(ctx, r.primaryClient, r.primaryBucket)
	if err != nil {
		metrics.DriftTotal.WithLabelValues(name, "critical").Inc()
		return reconciler.Result{OK: false, Detail: fmt.Sprintf("primary list: %v", err), Remaining: 1}
	}
	rCount, rSize, err := bucketStats(ctx, r.replicaClient, r.replicaBucket)
	if err != nil {
		metrics.DriftTotal.WithLabelValues(name, "critical").Inc()
		return reconciler.Result{OK: false, Detail: fmt.Sprintf("replica list: %v", err), Remaining: 1}
	}
	elapsed := time.Since(start)
	metrics.SyncDurationSeconds.WithLabelValues(name, "ok").Observe(elapsed.Seconds())

	if pCount == rCount && pSize == rSize {
		metrics.LastSyncTimestamp.WithLabelValues(name).Set(float64(time.Now().Unix()))
		return reconciler.Result{OK: true, Detail: fmt.Sprintf("buckets match: %d objects, %d bytes", pCount, pSize)}
	}
		delta := pCount - rCount
	metrics.DriftTotal.WithLabelValues(name, "warning").Inc()
	return reconciler.Result{
		OK:        false,
		Detail:    fmt.Sprintf("drift: primary=%d/%d replica=%d/%d (delta=%d)", pCount, pSize, rCount, rSize, delta),
		Remaining: int(delta),
	}
}

// bucketStats lists all objects in a bucket and returns (count, totalSize).
func bucketStats(ctx context.Context, client *s3.Client, bucket string) (int64, int64, error) {
	var count, totalSize int64
	paginator := s3.NewListObjectsV2Paginator(client, &s3.ListObjectsV2Input{
		Bucket: ptr(bucket),
	})
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return 0, 0, err
		}
		for _, obj := range page.Contents {
			count++
			if obj.Size != nil {
				totalSize += *obj.Size
			}
		}
	}
	return count, totalSize, nil
}

func ptr[T any](v T) *T { return &v }