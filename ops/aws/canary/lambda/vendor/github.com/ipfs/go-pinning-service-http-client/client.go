package go_pinning_service_http_client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/pkg/errors"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-pinning-service-http-client/openapi"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"

	logging "github.com/ipfs/go-log/v2"
)

var logger = logging.Logger("pinning-service-http-client")

const UserAgent = "go-pinning-service-http-client"

type Client struct {
	client *openapi.APIClient
}

func NewClient(url, bearerToken string) *Client {
	config := openapi.NewConfiguration()
	config.UserAgent = UserAgent
	bearer := fmt.Sprintf("Bearer %s", bearerToken)
	config.AddDefaultHeader("Authorization", bearer)
	config.Servers = openapi.ServerConfigurations{
		openapi.ServerConfiguration{
			URL: url,
		},
	}

	return &Client{client: openapi.NewAPIClient(config)}
}

// TODO: We should probably make sure there are no duplicates sent
type lsSettings struct {
	cids   []string
	name   string
	status []Status
	before *time.Time
	after  *time.Time
	limit  *int32
	meta   map[string]string
}

type LsOption func(options *lsSettings) error

var PinOpts = pinOpts{}

type pinOpts struct {
	pinLsOpts
	pinAddOpts
}

type pinLsOpts struct{}

func (pinLsOpts) FilterCIDs(cids ...cid.Cid) LsOption {
	return func(options *lsSettings) error {
		enc := getCIDEncoder()
		for _, c := range cids {
			options.cids = append(options.cids, c.Encode(enc))
		}
		return nil
	}
}

const maxNameSize = 255

func (pinLsOpts) FilterName(name string) LsOption {
	return func(options *lsSettings) error {
		if len(name) > maxNameSize {
			return fmt.Errorf("name cannot be longer than %d", maxNameSize)
		}
		options.name = name
		return nil
	}
}

func (pinLsOpts) FilterStatus(statuses ...Status) LsOption {
	return func(options *lsSettings) error {
		for _, s := range statuses {
			valid := false
			for _, existing := range validStatuses {
				if existing == s {
					valid = true
					break
				}
			}
			if !valid {
				return fmt.Errorf("invalid status %s", s)
			}
		}
		options.status = append(options.status, statuses...)
		return nil
	}
}

func (pinLsOpts) FilterBefore(t time.Time) LsOption {
	return func(options *lsSettings) error {
		options.before = &t
		return nil
	}
}

func (pinLsOpts) FilterAfter(t time.Time) LsOption {
	return func(options *lsSettings) error {
		options.after = &t
		return nil
	}
}

const recordLimit = 1000
const defaultLimit = 10

func (pinLsOpts) Limit(limit int) LsOption {
	return func(options *lsSettings) error {
		if limit > recordLimit {
			return fmt.Errorf("limit exceeded maximum record limit of %d", recordLimit)
		}
		limitCasted := int32(limit)
		options.limit = &limitCasted
		return nil
	}
}

func (pinLsOpts) LsMeta(meta map[string]string) LsOption {
	return func(options *lsSettings) error {
		options.meta = meta
		return nil
	}
}

type pinResults = openapi.PinResults

func (c *Client) Ls(ctx context.Context, opts ...LsOption) (chan PinStatusGetter, chan error) {
	res := make(chan PinStatusGetter, 1)
	errs := make(chan error, 1)

	settings := new(lsSettings)
	for _, o := range opts {
		if err := o(settings); err != nil {
			close(res)
			errs <- err
			close(errs)
			return res, errs
		}
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				var err error
				switch x := r.(type) {
				case string:
					err = fmt.Errorf("unexpected error while listing remote pins: %s", x)
				case error:
					err = fmt.Errorf("unexpected error while listing remote pins: %w", x)
				default:
					err = errors.New("unknown panic while listing remote pins")
				}
				errs <- err
			}
			close(errs)
			close(res)
		}()

		for {
			pinRes, err := c.lsInternal(ctx, settings)
			if err != nil {
				errs <- err
				return
			}

			results := pinRes.GetResults()
			for _, r := range results {
				select {
				case res <- &pinStatusObject{r}:
				case <-ctx.Done():
					errs <- ctx.Err()
					return
				}
			}

			batchSize := len(results)
			if int(pinRes.Count) == batchSize {
				// no more batches
				return
			}

			// Better DX/UX for cases like https://github.com/application-research/estuary/issues/124
			if batchSize == 0 && int(pinRes.Count) != 0 {
				errs <- fmt.Errorf("invalid pinning service response: PinResults.count=%d but no PinResults.results", int(pinRes.Count))
				return
			}

			oldestResult := results[batchSize-1]
			settings.before = &oldestResult.Created
		}
	}()

	return res, errs
}

func (c *Client) LsSync(ctx context.Context, opts ...LsOption) ([]PinStatusGetter, error) {
	resCh, errCh := c.Ls(ctx, opts...)

	var res []PinStatusGetter
	for r := range resCh {
		res = append(res, r)
	}

	return res, <-errCh
}

// Manual version of Ls that returns a single batch of results and int with total count
func (c *Client) LsBatchSync(ctx context.Context, opts ...LsOption) ([]PinStatusGetter, int, error) {
	var res []PinStatusGetter

	settings := new(lsSettings)
	for _, o := range opts {
		if err := o(settings); err != nil {
			return nil, 0, err
		}
	}

	pinRes, err := c.lsInternal(ctx, settings)
	if err != nil {
		return nil, 0, err
	}

	results := pinRes.GetResults()
	for _, r := range results {
		res = append(res, &pinStatusObject{r})
	}

	return res, int(pinRes.Count), nil
}

func (c *Client) lsInternal(ctx context.Context, settings *lsSettings) (pinResults, error) {
	getter := c.client.PinsApi.PinsGet(ctx)
	if len(settings.cids) > 0 {
		getter = getter.Cid(settings.cids)
	}
	if len(settings.status) > 0 {
		statuses := make([]openapi.Status, len(settings.status))
		for i := 0; i < len(statuses); i++ {
			statuses[i] = openapi.Status(settings.status[i])
		}
		getter = getter.Status(statuses)
	}
	if settings.limit == nil {
		getter = getter.Limit(defaultLimit)
	} else {
		getter = getter.Limit(*settings.limit)
	}
	if len(settings.name) > 0 {
		getter = getter.Name(settings.name)
	}
	if settings.before != nil {
		getter = getter.Before(*settings.before)
	}
	if settings.after != nil {
		getter = getter.After(*settings.after)
	}
	if settings.meta != nil {
		getter = getter.Meta(settings.meta)
	}

	// TODO: Ignoring HTTP Response OK?
	results, httpresp, err := getter.Execute()
	if err != nil {
		err := httperr(httpresp, err)
		return pinResults{}, err
	}

	return results, nil
}

// TODO: We should probably make sure there are no duplicates sent
type addSettings struct {
	name    string
	origins []string
	meta    map[string]string
}

type AddOption func(options *addSettings) error

type pinAddOpts struct{}

func (pinAddOpts) WithName(name string) AddOption {
	return func(options *addSettings) error {
		if len(name) > maxNameSize {
			return fmt.Errorf("name cannot be longer than %d", maxNameSize)
		}
		options.name = name
		return nil
	}
}

func (pinAddOpts) WithOrigins(origins ...multiaddr.Multiaddr) AddOption {
	return func(options *addSettings) error {
		for _, o := range origins {
			options.origins = append(options.origins, o.String())
		}
		return nil
	}
}

func (pinAddOpts) AddMeta(meta map[string]string) AddOption {
	return func(options *addSettings) error {
		options.meta = meta
		return nil
	}
}

func (c *Client) Add(ctx context.Context, cid cid.Cid, opts ...AddOption) (PinStatusGetter, error) {
	settings := new(addSettings)
	for _, o := range opts {
		if err := o(settings); err != nil {
			return nil, err
		}
	}

	adder := c.client.PinsApi.PinsPost(ctx)
	p := openapi.Pin{
		Cid: cid.Encode(getCIDEncoder()),
	}

	if len(settings.origins) > 0 {
		p.SetOrigins(settings.origins)
	}
	if settings.meta != nil {
		p.SetMeta(settings.meta)
	}
	if len(settings.name) > 0 {
		p.SetName(settings.name)
	}

	result, httpresp, err := adder.Pin(p).Execute()
	if err != nil {
		err := httperr(httpresp, err)
		return nil, err
	}

	return &pinStatusObject{result}, nil
}

func (c *Client) GetStatusByID(ctx context.Context, pinID string) (PinStatusGetter, error) {
	getter := c.client.PinsApi.PinsRequestidGet(ctx, pinID)
	result, httpresp, err := getter.Execute()
	if err != nil {
		err := httperr(httpresp, err)
		return nil, err
	}

	return &pinStatusObject{result}, nil
}

func (c *Client) DeleteByID(ctx context.Context, pinID string) error {
	deleter := c.client.PinsApi.PinsRequestidDelete(ctx, pinID)
	httpresp, err := deleter.Execute()
	if err != nil {
		err := httperr(httpresp, err)
		return err
	}
	return nil
}

func (c *Client) Replace(ctx context.Context, pinID string, cid cid.Cid, opts ...AddOption) (PinStatusGetter, error) {
	settings := new(addSettings)
	for _, o := range opts {
		if err := o(settings); err != nil {
			return nil, err
		}
	}

	adder := c.client.PinsApi.PinsRequestidPost(ctx, pinID)
	p := openapi.Pin{
		Cid: cid.Encode(getCIDEncoder()),
	}

	if len(settings.origins) > 0 {
		p.SetOrigins(settings.origins)
	}
	if settings.meta != nil {
		p.SetMeta(settings.meta)
	}
	if len(settings.name) > 0 {
		p.SetName(settings.name)
	}

	result, httpresp, err := adder.Pin(p).Execute()
	if err != nil {
		err := httperr(httpresp, err)
		return nil, err
	}

	return &pinStatusObject{result}, nil
}

func getCIDEncoder() multibase.Encoder {
	enc, err := multibase.NewEncoder(multibase.Base32)
	if err != nil {
		panic(err)
	}
	return enc
}

func httperr(resp *http.Response, e error) error {
	oerr, ok := e.(openapi.GenericOpenAPIError)
	if ok {
		ferr, ok := oerr.Model().(openapi.Failure)
		if ok {
			return errors.Wrapf(e, "reason: %q, details: %q", ferr.Error.GetReason(), ferr.Error.GetDetails())
		}
	}

	if resp == nil {
		return errors.Wrapf(e, "empty response from remote pinning service")
	}

	return errors.Wrapf(e, "remote pinning service returned http error %d", resp.StatusCode)
}
