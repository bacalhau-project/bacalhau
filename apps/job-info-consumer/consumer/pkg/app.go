package pkg

import (
	"context"

	"github.com/libp2p/go-libp2p/core/host"
)

type ApplicationParams struct {
	PostgresDatastoreParams
	Libp2pHost host.Host
}

// Application is the main application struct.
// It creates postgres datastore and consumer, and connects them together.
type Application struct {
	consumer  *Consumer
	datastore *PostgresDatastore
}

func NewApplication(params ApplicationParams) (*Application, error) {
	datastore, err := NewPostgresDatastore(params.PostgresDatastoreParams)
	if err != nil {
		return nil, err
	}
	consumer := NewConsumer(ConsumerParams{
		Libp2pHost: params.Libp2pHost,
		Datastore:  datastore,
	})
	if err != nil {
		return nil, err
	}
	return &Application{
		consumer:  consumer,
		datastore: datastore,
	}, nil
}

// Start starts the application by starting the gossipsub consumer.
func (a *Application) Start(ctx context.Context) error {
	return a.consumer.Start(ctx)
}

// Stop stops the application by stopping the gossipsub consumer and closing the datastore.
func (a *Application) Stop(ctx context.Context) error {
	err := a.consumer.Stop(ctx)
	if err != nil {
		return err
	}
	return a.datastore.Close()
}
