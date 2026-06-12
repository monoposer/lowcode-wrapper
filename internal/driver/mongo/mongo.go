package mongo

import (
	"context"
	"fmt"
	"strings"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolMongo, New)
}

type Driver struct {
	client *mongo.Client
	db     string
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.MongoServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	uri := strings.TrimSpace(opts.URI)
	if uri == "" && cred != nil {
		uri = strings.TrimSpace(fmt.Sprint(cred["uri"]))
	}
	if uri == "" {
		return nil, fmt.Errorf("mongo server %q requires options.uri or credential.uri", srv.Name)
	}
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}
	if err := client.Ping(ctx, nil); err != nil {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo ping: %w", err)
	}
	db := strings.TrimSpace(opts.Database)
	if db == "" && cred != nil {
		db = strings.TrimSpace(fmt.Sprint(cred["database"]))
	}
	if db == "" {
		_ = client.Disconnect(ctx)
		return nil, fmt.Errorf("mongo server %q requires options.database", srv.Name)
	}
	return &Driver{client: client, db: db}, nil
}

func (d *Driver) Close() error {
	if d.client != nil {
		return d.client.Disconnect(context.Background())
	}
	return nil
}

func (d *Driver) collection(resolved *models.ResolvedTable) (*mongo.Collection, error) {
	topts, _ := models.ParseServerOptions[models.MongoTableOptions](resolved.Table.Options)
	name := strings.TrimSpace(topts.Collection)
	if name == "" {
		name = models.RemoteTableName(resolved.Table)
	}
	if name == "" {
		return nil, fmt.Errorf("mongo: collection name required")
	}
	return d.client.Database(d.db).Collection(name), nil
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	coll, err := d.collection(req.Resolved)
	if err != nil {
		return nil, err
	}
	filter := filtersToBSON(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	findOpts := options.Find()
	proj := projectBSON(req.Select)
	if proj != nil {
		findOpts.SetProjection(proj)
	}
	if req.Limit > 0 {
		findOpts.SetLimit(int64(req.Limit))
	}
	if req.Offset > 0 {
		findOpts.SetSkip(int64(req.Offset))
	}
	if len(req.Order) > 0 {
		sort := bson.D{}
		for _, o := range req.Order {
			dir := 1
			if o.Desc {
				dir = -1
			}
			sort = append(sort, bson.E{Key: o.Column, Value: dir})
		}
		findOpts.SetSort(sort)
	}
	cur, err := coll.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []map[string]any
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	return postgrest.MapRowsFromRemote(docs, req.Resolved.Columns), nil
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	coll, err := d.collection(req.Resolved)
	if err != nil {
		return nil, err
	}
	doc := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	res, err := coll.InsertOne(ctx, doc)
	if err != nil {
		return nil, err
	}
	if req.PreferRepresentation {
		doc["_id"] = res.InsertedID
		return postgrest.MapRowsFromRemote([]map[string]any{doc}, req.Resolved.Columns)[0], nil
	}
	return nil, nil
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	coll, err := d.collection(req.Resolved)
	if err != nil {
		return 0, err
	}
	filter := filtersToBSON(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	update := bson.M{"$set": postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)}
	res, err := coll.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return int(res.ModifiedCount), nil
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	coll, err := d.collection(req.Resolved)
	if err != nil {
		return false, nil, err
	}
	doc := postgrest.MapRowToRemote(req.Row, req.Resolved.Columns)
	filter := bson.M{}
	for _, k := range req.Resolved.Table.KeyColumns {
		if v, ok := doc[k]; ok {
			filter[k] = v
		}
	}
	if len(filter) == 0 {
		return false, nil, fmt.Errorf("mongo upsert requires key_columns")
	}
	opts := options.Update().SetUpsert(true)
	res, err := coll.UpdateOne(ctx, filter, bson.M{"$set": doc}, opts)
	if err != nil {
		return false, nil, err
	}
	created := res.UpsertedCount > 0
	if req.PreferRepresentation {
		return created, postgrest.MapRowsFromRemote([]map[string]any{doc}, req.Resolved.Columns)[0], nil
	}
	return created, nil, nil
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	coll, err := d.collection(req.Resolved)
	if err != nil {
		return 0, err
	}
	filter := filtersToBSON(postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	res, err := coll.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return int(res.DeletedCount), nil
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	return nil, driver.OpNotSupported(driver.OpInvoke, models.ProtocolMongo)
}

func filtersToBSON(filters []postgrest.Filter) bson.M {
	if len(filters) == 0 {
		return bson.M{}
	}
	m := bson.M{}
	for _, f := range filters {
		switch f.Op {
		case postgrest.OpEq:
			m[f.Column] = f.Value
		case postgrest.OpNeq:
			m[f.Column] = bson.M{"$ne": f.Value}
		case postgrest.OpGt:
			m[f.Column] = bson.M{"$gt": f.Value}
		case postgrest.OpGte:
			m[f.Column] = bson.M{"$gte": f.Value}
		case postgrest.OpLt:
			m[f.Column] = bson.M{"$lt": f.Value}
		case postgrest.OpLte:
			m[f.Column] = bson.M{"$lte": f.Value}
		case postgrest.OpIn:
			parts := strings.Split(f.Value, ",")
			m[f.Column] = bson.M{"$in": parts}
		default:
			m[f.Column] = f.Value
		}
	}
	return m
}

func projectBSON(cols []string) bson.M {
	if len(cols) == 0 {
		return nil
	}
	p := bson.M{"_id": 0}
	for _, c := range cols {
		p[c] = 1
	}
	return p
}
