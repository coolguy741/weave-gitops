package server

import (
	"context"
	"fmt"

	pb "github.com/weaveworks/weave-gitops/pkg/api/core"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	corev1 "k8s.io/api/core/v1"
)

func (cs *coreServer) ListFluxEvents(ctx context.Context, msg *pb.ListFluxEventsRequest) (*pb.ListFluxEventsResponse, error) {
	k8s, err := cs.k8s.Client(ctx)
	if err != nil {
		return nil, doClientError(err)
	}

	l := &corev1.EventList{}
	if err := list(ctx, k8s, temporarilyEmptyAppName, msg.Namespace, l); err != nil {
		return nil, fmt.Errorf("could not get events: %w", err)
	}

	if msg.InvolvedObject == nil {
		return nil, status.Errorf(codes.InvalidArgument, "bad request: no object was specified")
	}

	events := []*pb.Event{}

	for _, e := range l.Items {
		if isObjectReference(e.InvolvedObject, msg.InvolvedObject) {
			events = append(events, &pb.Event{
				Type:      e.Type,
				Component: e.Source.Component,
				Name:      e.ObjectMeta.Name,
				Reason:    e.Reason,
				Message:   e.Message,
				Timestamp: int32(e.LastTimestamp.Unix()),
				Host:      e.Source.Host,
			})
		}
	}

	return &pb.ListFluxEventsResponse{Events: events}, nil
}

func isObjectReference(ref corev1.ObjectReference, msg *pb.ObjectReference) bool {
	if ref.Kind == msg.Kind && ref.Name == msg.Name && ref.Namespace == msg.Namespace {
		return true
	}

	return false
}
