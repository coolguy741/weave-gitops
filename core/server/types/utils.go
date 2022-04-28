package types

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2beta1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1beta2"
	pb "github.com/weaveworks/weave-gitops/pkg/api/core"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getSourceKind(kind string) pb.SourceRef_SourceKind {
	switch kind {
	case sourcev1.GitRepositoryKind:
		return pb.SourceRef_GitRepository
	case sourcev1.HelmRepositoryKind:
		return pb.SourceRef_HelmRepository
	case sourcev1.BucketKind:
		return pb.SourceRef_Bucket
	default:
		return -1
	}
}

func mapConditions(conditions []metav1.Condition) []*pb.Condition {
	out := []*pb.Condition{}

	for _, c := range conditions {
		out = append(out, &pb.Condition{
			Type:      c.Type,
			Status:    string(c.Status),
			Reason:    c.Reason,
			Message:   c.Message,
			Timestamp: c.LastTransitionTime.String(),
		})
	}

	return out
}

func lastUpdatedAt(obj interface{}) string {
	switch s := obj.(type) {
	case *sourcev1.GitRepository:
		if s.Status.Artifact != nil {
			return s.Status.Artifact.LastUpdateTime.String()
		}
	case *sourcev1.Bucket:
		if s.Status.Artifact != nil {
			return s.Status.Artifact.LastUpdateTime.String()
		}
	case *sourcev1.HelmChart:
		if s.Status.Artifact != nil {
			return s.Status.Artifact.LastUpdateTime.String()
		}
	case *sourcev1.HelmRepository:
		if s.Status.Artifact != nil {
			return s.Status.Artifact.LastUpdateTime.String()
		}
	case *helmv2.HelmRelease:
		return s.Status.LastHandledReconcileAt
	}

	return ""
}

func durationToInterval(duration metav1.Duration) *pb.Interval {
	return &pb.Interval{
		Hours:   int64(duration.Hours()),
		Minutes: int64(duration.Minutes()) % 60,
		Seconds: int64(duration.Seconds()) % 60,
	}
}
