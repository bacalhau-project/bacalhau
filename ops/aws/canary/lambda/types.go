package main

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
)

type Event struct {
	Action string `json:"action"`
}

type Handler func(ctx context.Context, client *publicapi.APIClient) error
