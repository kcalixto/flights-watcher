package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go/ptr"
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
		Value: fmt.Sprintf("%s#%s",
			FLIGHT,
			flight.DepartueToken,
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
		Value: fmt.Sprintf("%s#%s",
			FLIGHT,
			LOWEST_PRICE,
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

func (r *FlightsWatcherRepository) GetLowestPrice(ctx context.Context) (*Flight, error) {
	res, err := r.conn.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: ptr.String(r.tableName),
		Key: map[string]types.AttributeValue{
			PK: &types.AttributeValueMemberS{
				Value: fmt.Sprintf("%s#%s",
					FLIGHT,
					LOWEST_PRICE,
				),
			},
			SK: &types.AttributeValueMemberS{
				Value: FLIGHT,
			},
		},
	})
	if err != nil {
		slog.Error(fmt.Sprintf("error querying: %s", err.Error()))
		return nil, err
	}

	if res.Item == nil {
		return nil, nil
	}

	var flight Flight
	err = attributevalue.UnmarshalMap(res.Item, &flight)
	if err != nil {
		slog.Error(fmt.Sprintf("error unmarshalling: %s", err.Error()))
		return nil, err
	}

	return &flight, nil
}
