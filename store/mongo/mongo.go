package mongo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

type Mongo struct {
	Client *mongo.Client
	config *Config
}

type Option func(*Mongo)

func New(c *Config, opts ...Option) (*Mongo, error) {
	m := &Mongo{
		config: c,
	}

	if err := m.config.init(); err != nil {
		return m, err
	}

	for _, opt := range opts {
		opt(m)
	}

	return m.new()
}

func (m *Mongo) new() (*Mongo, error) {
	serverApi := options.ServerAPI(options.ServerAPIVersion1)
	bsonOpts := &options.BSONOptions{
		UseJSONStructTags: true,
		NilSliceAsEmpty:   true,
	}
	opts := options.Client().ApplyURI(m.config.uri()).SetServerAPIOptions(serverApi).SetBSONOptions(bsonOpts)
	client, err := mongo.Connect(context.TODO(), opts)
	if err != nil {
		return nil, err
	}
	m.Client = client

	if err := m.Ping(); err != nil {
		return nil, err
	}

	return m, nil
}

func (m *Mongo) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.config.Timeout)*time.Second)
	defer cancel()

	return m.Client.Ping(ctx, readpref.Primary())
}

func (m *Mongo) Close() error {
	if m.Client == nil {
		return nil
	}

	return m.Client.Disconnect(context.TODO())
}
