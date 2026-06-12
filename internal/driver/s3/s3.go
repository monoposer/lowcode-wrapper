package s3

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/monoposer/dataspan/internal/driver"
	"github.com/monoposer/dataspan/internal/models"
	"github.com/monoposer/dataspan/internal/postgrest"
)

func init() {
	driver.Register(models.ProtocolS3, New)
}

type Driver struct {
	client *s3.Client
	bucket string
}

func New(ctx context.Context, srv models.Server, cred map[string]any) (driver.Driver, error) {
	opts, err := models.ParseServerOptions[models.S3ServerOptions](srv.Options)
	if err != nil {
		return nil, err
	}
	bucket := strings.TrimSpace(opts.Bucket)
	if bucket == "" {
		return nil, fmt.Errorf("s3 server %q requires options.bucket", srv.Name)
	}
	cfg, err := loadAWSConfig(ctx, opts.Region, cred)
	if err != nil {
		return nil, err
	}
	return &Driver{
		client: s3.NewFromConfig(cfg),
		bucket: bucket,
	}, nil
}

func loadAWSConfig(ctx context.Context, region string, cred map[string]any) (aws.Config, error) {
	region = strings.TrimSpace(region)
	if region == "" && cred != nil {
		region = strings.TrimSpace(fmt.Sprint(cred["region"]))
	}
	loadOpts := []func(*awsconfig.LoadOptions) error{}
	if region != "" {
		loadOpts = append(loadOpts, awsconfig.WithRegion(region))
	}
	if cred != nil {
		id := strings.TrimSpace(fmt.Sprint(cred["accessKeyId"]))
		secret := strings.TrimSpace(fmt.Sprint(cred["secretAccessKey"]))
		if id != "" && secret != "" {
			token := strings.TrimSpace(fmt.Sprint(cred["sessionToken"]))
			loadOpts = append(loadOpts, awsconfig.WithCredentialsProvider(
				credentials.NewStaticCredentialsProvider(id, secret, token),
			))
		}
	}
	return awsconfig.LoadDefaultConfig(ctx, loadOpts...)
}

func (d *Driver) objectPrefix(resolved *models.ResolvedTable) (prefix, format string) {
	topts, _ := models.ParseServerOptions[models.S3TableOptions](resolved.Table.Options)
	prefix = strings.TrimSpace(topts.Prefix)
	if prefix == "" {
		prefix = models.RemoteTableName(resolved.Table)
	}
	format = strings.ToLower(strings.TrimSpace(topts.Format))
	return prefix, format
}

func (d *Driver) Select(ctx context.Context, req driver.SelectRequest) ([]map[string]any, error) {
	prefix, format := d.objectPrefix(req.Resolved)
	if format == "" && strings.Contains(prefix, ".") {
		format = extFormat(prefix)
	}
	if format != "" {
		rows, err := d.readObject(ctx, prefix, format)
		if err != nil {
			return nil, err
		}
		return d.finishSelect(rows, req), nil
	}
	out, err := d.client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{
		Bucket: aws.String(d.bucket),
		Prefix: aws.String(prefix),
	})
	if err != nil {
		return nil, err
	}
	var rows []map[string]any
	for _, obj := range out.Contents {
		if obj.Key == nil {
			continue
		}
		rows = append(rows, map[string]any{
			"key":  *obj.Key,
			"size": obj.Size,
		})
	}
	return d.finishSelect(rows, req), nil
}

func (d *Driver) readObject(ctx context.Context, key, format string) ([]map[string]any, error) {
	res, err := d.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(d.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	switch format {
	case "json":
		var arr []map[string]any
		if err := json.Unmarshal(data, &arr); err == nil {
			return arr, nil
		}
		var one map[string]any
		if err := json.Unmarshal(data, &one); err != nil {
			return nil, err
		}
		return []map[string]any{one}, nil
	case "ndjson", "jsonl":
		return parseNDJSON(data)
	default:
		return parseCSV(data)
	}
}

func (d *Driver) finishSelect(rows []map[string]any, req driver.SelectRequest) []map[string]any {
	rows = applyFilters(rows, postgrest.MapFilters(req.Filters, req.Resolved.Columns))
	rows = applySelectCols(rows, req.Select)
	if req.Offset > 0 && req.Offset < len(rows) {
		rows = rows[req.Offset:]
	} else if req.Offset >= len(rows) {
		return nil
	}
	if req.Limit > 0 && len(rows) > req.Limit {
		rows = rows[:req.Limit]
	}
	return postgrest.MapRowsFromRemote(rows, req.Resolved.Columns)
}

func (d *Driver) Insert(ctx context.Context, req driver.RowRequest) (map[string]any, error) {
	return nil, fmt.Errorf("s3 driver: insert not supported in phase 1")
}

func (d *Driver) Update(ctx context.Context, req driver.RowRequest) (int, error) {
	return 0, fmt.Errorf("s3 driver: update not supported in phase 1")
}

func (d *Driver) Upsert(ctx context.Context, req driver.RowRequest) (bool, map[string]any, error) {
	return false, nil, fmt.Errorf("s3 driver: upsert not supported in phase 1")
}

func (d *Driver) Delete(ctx context.Context, req driver.DeleteRequest) (int, error) {
	return 0, fmt.Errorf("s3 driver: delete not supported in phase 1")
}

func (d *Driver) Invoke(ctx context.Context, req driver.InvokeRequest) (any, error) {
	return nil, fmt.Errorf("s3 driver: invoke not supported")
}

func extFormat(key string) string {
	switch strings.ToLower(filepathExt(key)) {
	case ".csv":
		return "csv"
	case ".json":
		return "json"
	case ".ndjson", ".jsonl":
		return "ndjson"
	default:
		return ""
	}
}

func filepathExt(path string) string {
	if i := strings.LastIndex(path, "."); i >= 0 {
		return path[i:]
	}
	return ""
}

func parseCSV(data []byte) ([]map[string]any, error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		return nil, err
	}
	if len(records) < 1 {
		return nil, nil
	}
	header := records[0]
	var rows []map[string]any
	for _, rec := range records[1:] {
		row := make(map[string]any, len(header))
		for i, h := range header {
			if i < len(rec) {
				row[h] = rec[i]
			}
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func parseNDJSON(data []byte) ([]map[string]any, error) {
	lines := strings.Split(string(data), "\n")
	var rows []map[string]any
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			return nil, err
		}
		rows = append(rows, row)
	}
	return rows, nil
}

func applyFilters(rows []map[string]any, filters []postgrest.Filter) []map[string]any {
	if len(filters) == 0 {
		return rows
	}
	var out []map[string]any
	for _, row := range rows {
		ok := true
		for _, f := range filters {
			if fmt.Sprint(row[f.Column]) != f.Value && f.Op == postgrest.OpEq {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, row)
		}
	}
	return out
}

func applySelectCols(rows []map[string]any, cols []string) []map[string]any {
	if len(cols) == 0 {
		return rows
	}
	out := make([]map[string]any, len(rows))
	for i, row := range rows {
		m := make(map[string]any, len(cols))
		for _, c := range cols {
			if v, ok := row[c]; ok {
				m[c] = v
			}
		}
		out[i] = m
	}
	return out
}
