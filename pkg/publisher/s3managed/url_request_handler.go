package s3managed

import (
	"context"
	"fmt"
	"reflect"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/lib/envelope"
	"github.com/bacalhau-project/bacalhau/pkg/lib/ncl"
	"github.com/bacalhau-project/bacalhau/pkg/models/messages"
	"github.com/rs/zerolog/log"
)

type PreSignedURLRequestHandler struct {
	urlGenerator *PreSignedURLGenerator
}

func NewPreSignedURLRequestHandler(urlGenerator *PreSignedURLGenerator) *PreSignedURLRequestHandler {
	return &PreSignedURLRequestHandler{
		urlGenerator: urlGenerator,
	}
}

func (rh *PreSignedURLRequestHandler) HandleRequest(ctx context.Context, message *envelope.Message) (*envelope.Message, error) {
	request, ok := message.Payload.(*messages.ManagedPublisherPreSignURLRequest)
	if !ok {
		return nil, envelope.NewErrUnexpectedPayloadType("ManagedPublisherPreSignURLRequest", reflect.TypeOf(message.Payload).String())
	}

	if !rh.urlGenerator.IsInstalled() {
		return nil, bacerrors.New("Managed S3 publisher is not available").
			WithCode(bacerrors.BadRequestError)
	}

	if request.JobID == "" || request.ExecutionID == "" {
		return nil, envelope.NewErrBadPayload("JobID and ExecutionID must be provided")
	}

	log.Ctx(ctx).Debug().
		Str("job_id", request.JobID).
		Str("execution_id", request.ExecutionID).
		Msg("Received a request to generate pre-signed URL for S3 managed publisher")

	url, err := rh.urlGenerator.GeneratePreSignedPutURL(ctx, request.JobID, request.ExecutionID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate pre-signed PUT URL: %w", err)
	}
	return envelope.NewMessage(messages.ManagedPublisherPreSignURLResponse{
		JobID:        request.JobID,
		ExecutionID:  request.ExecutionID,
		PreSignedURL: url,
	}).WithMetadataValue(envelope.KeyMessageType, messages.ManagedPublisherPreSignURLResponseType), nil
}

var _ ncl.RequestHandler = (*PreSignedURLRequestHandler)(nil)
