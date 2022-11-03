package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"time"
)

const mongoOperationTimeout = 5 * time.Second

func newMongoServer() (*mongoClient, error) {
	ms := &mongoClient{}
	return ms, nil
}

type mongoClient struct {
	client *mongo.Client
}

func (m *mongoClient) Connect(uri string) error {
	log.Infof("connect to mongodb[%s]", uri)
	ctx, cancel := context.WithTimeout(context.TODO(), 3*time.Second)
	defer cancel()

	var err error
	m.client, err = mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return err
	}
	return m.client.Ping(ctx, readpref.Primary())
}

func (m *mongoClient) Close() {
	if m.client != nil {
		if err := m.client.Disconnect(nil); err != nil {
			log.Error("mongodb Disconnect occur error: ", err.Error())
			return
		}
		m.client = nil
	}
	log.Info("mongo server closed")
}

func (m *mongoClient) GetCollection(database, collection string) *mongo.Collection {
	return m.client.Database(database).Collection(collection)
}

func (m *mongoClient) FindOne(database, collection string, filter interface{}) (bson.M, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	var result bson.M
	err := m.GetCollection(database, collection).FindOne(ctx, filter).Decode(&result)
	if err == mongo.ErrNoDocuments {
		err = nil
	}
	return result, err
}

func (m *mongoClient) FindMany(database, collection string, filter interface{}, opts *options.FindOptions) ([]bson.M, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	var results []bson.M
	cursor, err := m.GetCollection(database, collection).Find(ctx, filter, opts)
	if err != nil {
		return results, err
	}
	defer cursor.Close(ctx)
	err = cursor.All(context.TODO(), &results)

	return results, err
}

func (m *mongoClient) InsertOne(database, collection string, data interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()
	_, err := m.GetCollection(database, collection).InsertOne(ctx, data)
	return err
}

func (m *mongoClient) InsertMany(database, collection string, dataArray []interface{}) (*mongo.InsertManyResult, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	return m.GetCollection(database, collection).InsertMany(ctx, dataArray)
}

func (m *mongoClient) UpdateOne(database, collection string, filter interface{}, data interface{}, opts ...*options.UpdateOptions) error {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	_, err := m.GetCollection(database, collection).UpdateOne(ctx, filter, bson.D{{"$set", data}}, opts...)
	return err
}

func (m *mongoClient) UpdateMany(database, collection string, filter interface{}, data interface{}, opts ...*options.UpdateOptions) (*mongo.UpdateResult, error) {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	return m.GetCollection(database, collection).UpdateMany(ctx, filter, bson.D{{"$set", data}}, opts...)
}

func (m *mongoClient) ReplaceOne(database, collection string, filter interface{}, data interface{}, opts ...*options.ReplaceOptions) error {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	_, err := m.GetCollection(database, collection).ReplaceOne(ctx, filter, data, opts...)
	return err
}

func (m *mongoClient) DeleteOne(database, collection string, filter interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	_, err := m.GetCollection(database, collection).DeleteOne(ctx, filter)
	return err
}

func (m *mongoClient) DeleteMany(database, collection string, filter interface{}) error {
	ctx, cancel := context.WithTimeout(context.TODO(), mongoOperationTimeout)
	defer cancel()

	_, err := m.GetCollection(database, collection).DeleteMany(ctx, filter)
	return err
}
