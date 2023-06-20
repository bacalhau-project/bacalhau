package queue

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/samber/lo"
	spb "go.etcd.io/etcd/api/v3/mvccpb"
	etcd_client "go.etcd.io/etcd/client/v3"
)

// NewSequentialKV creates a new sequential entry in the provided prefix and
// returns the newly created key and an error (if any). Based on etcd recipes
// from
// https://github.com/etcd-io/etcd/blob/main/client/v3/experimental/recipes
func NewSequentialKV(
	ctx context.Context,
	client etcd_client.KV,
	prefix, val string) (key string, e error) {
	response, err := client.Get(ctx, prefix, etcd_client.WithLastKey()...)
	if err != nil {
		return "", err
	}

	// Increment the last key if any found
	sequence := 0
	if len(response.Kvs) != 0 {
		fields := strings.Split(string(response.Kvs[0].Key), "/")
		_, serr := fmt.Sscanf(fields[len(fields)-1], "%d", &sequence)
		if serr != nil {
			return "", serr
		}
		sequence++
	}
	newKey := fmt.Sprintf("%s/%016d", prefix, sequence)

	// Scope the Put request to avoid changes whilst we are doing this work, if the
	// comparison fails the transaction we can retry
	baseKey := "__" + prefix
	cmp := etcd_client.Compare(etcd_client.ModRevision(baseKey), "<", response.Header.Revision+1)
	reqPrefix := etcd_client.OpPut(baseKey, "")
	reqnewKey := etcd_client.OpPut(newKey, val)

	txn := client.Txn(ctx)
	txnResponse, err := txn.If(cmp).Then(reqPrefix, reqnewKey).Commit()
	if err != nil {
		return "", err
	}

	// Retry if the transaction failed (somebody beat us to it with a write)
	if !txnResponse.Succeeded {
		return NewSequentialKV(ctx, client, prefix, val)
	}

	return newKey, nil
}

func ClaimFirstKey(kv etcd_client.KV, kvs []*spb.KeyValue) (*spb.KeyValue, error) {
	for _, k := range kvs {
		ok, err := DeleteRevKey(kv, string(k.Key), k.ModRevision)
		if err != nil {
			return nil, err
		} else if ok {
			return k, nil
		}
	}
	return nil, nil
}

// DeleteRevKey deletes a key by revision, returning false if key is missing
func DeleteRevKey(kv etcd_client.KV, key string, rev int64) (succeeded bool, err error) {
	cmp := etcd_client.Compare(etcd_client.ModRevision(key), "=", rev)
	req := etcd_client.OpDelete(key)
	txnResponse, err := kv.Txn(context.TODO()).If(cmp).Then(req).Commit()
	if err != nil {
		return false, err
	} else if !txnResponse.Succeeded {
		return false, nil
	}
	return true, nil
}

func WaitPrefixEvents(ctx context.Context,
	c *etcd_client.Client, prefix string, rev int64, eventType spb.Event_EventType) (*etcd_client.Event, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wc := c.Watch(ctx, prefix, etcd_client.WithPrefix(), etcd_client.WithRev(rev))
	if wc == nil {
		return nil, errors.New("unable to watch prefix from revision")
	}

	return waitEvents(wc, eventType), nil
}

func waitEvents(wc etcd_client.WatchChan, eventType spb.Event_EventType) *etcd_client.Event {
	for wresp := range wc {
		event, ok := lo.Find(wresp.Events, func(e *etcd_client.Event) bool {
			return e.Type == eventType
		})

		if ok {
			return event
		}
	}

	return nil
}
