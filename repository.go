package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/ptr"
	"github.com/google/uuid"
)

type FlightsWatcherRepository struct {
	conn      *dynamodb.Client
	tableName string
}

const (
	PK           = "pk"
	SK           = "sk"
	FLIGHT       = "FLIGHT"
	LOWEST_PRICE = "LOWEST_PRICE"
	EXPIRES_AT   = "expires_at"
	SK_INDEX     = "sk-index"
)

func NewFlightsWatcherRepository(conn *dynamodb.Client) *FlightsWatcherRepository {
	tableName := os.Getenv("TABLE_NAME")
	if tableName == "" {
		panic("TABLE_NAME is required")
	}
	return &FlightsWatcherRepository{conn: conn, tableName: tableName}
}

func (r *FlightsWatcherRepository) SaveFlight(ctx context.Context, flight *Flight) error {
	attr, err := attributevalue.MarshalMap(flight)
	if err != nil {
		slog.Error(fmt.Sprintf("error marshalling flight: %s", err.Error()))
		return err
	}

	attr[PK] = &types.AttributeValueMemberS{
		Value: fmt.Sprintf("%s#%s#%s",
			time.Now().Format(time.RFC3339),
			FLIGHT,
			uuid.New().String(),
		),
	}
	attr[SK] = &types.AttributeValueMemberS{
		Value: FLIGHT,
	}

	_, err = r.conn.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: ptr.String(r.tableName),
		Item:      attr,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error putting item: %s", err.Error()))
		return err
	}

	return nil
}

func (r *FlightsWatcherRepository) SaveLowestPrice(ctx context.Context, flight *Flight) error {
	attr, err := attributevalue.MarshalMap(flight)
	if err != nil {
		slog.Error(fmt.Sprintf("error marshalling flight: %s", err.Error()))
		return err
	}

	attr[PK] = &types.AttributeValueMemberS{
		Value: fmt.Sprintf("%s#%s#%s",
			time.Now().Format(time.RFC3339),
			FLIGHT,
			uuid.New().String(),
		),
	}
	attr[SK] = &types.AttributeValueMemberS{
		Value: LOWEST_PRICE,
	}

	_, err = r.conn.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: ptr.String(r.tableName),
		Item:      attr,
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error putting item: %s", err.Error()))
		return err
	}

	return nil
}

func (r *FlightsWatcherRepository) GetLowestPrice(ctx context.Context) (*Flight, error) {
	res, err := r.conn.Query(ctx, &dynamodb.QueryInput{
		TableName:              ptr.String(r.tableName),
		IndexName:              ptr.String(SK_INDEX),
		KeyConditionExpression: ptr.String("#sk = :sk"),
		ExpressionAttributeNames: map[string]string{
			"#sk": SK,
		},
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":sk": &types.AttributeValueMemberS{
				Value: LOWEST_PRICE,
			},
		},
		Limit: ptr.Int32(1),
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error querying: %s", err.Error()))
		return nil, err
	}

	if len(res.Items) == 0 {
		return nil, nil
	}

	var flight Flight
	err = attributevalue.UnmarshalMap(res.Items[0], &flight)
	if err != nil {
		slog.Error(fmt.Sprintf("error unmarshalling: %s", err.Error()))
		return nil, err
	}

	return &flight, nil
}
