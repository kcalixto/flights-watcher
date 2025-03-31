package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ses"

	g "github.com/serpapi/google-search-results-golang"
)

var repo *FlightsWatcherRepository
var mail *Mail

func initDependencies() error {
	awsconfigv2, err := config.LoadDefaultConfig(context.Background(), config.WithRegion("sa-east-1"))
	if err != nil {
		slog.Error(err.Error())
		return err
	}
	awssess, err := session.NewSession(&aws.Config{
		Region: aws.String("sa-east-1"),
	})
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	repo = NewFlightsWatcherRepository(dynamodb.NewFromConfig(awsconfigv2))
	mail = NewMail(ses.New(awssess))

	return nil
}

func handler(ctx context.Context) (err error) {
	parameter := map[string]string{
		"engine":        "google_flights",
		"hl":            "en",
		"gl":            "br",
		"departure_id":  "CGH",
		"arrival_id":    "FOR",
		"outbound_date": "2025-12-26",
		"return_date":   "2026-01-06",
		"currency":      "BRL",
		"sort_by":       "2",
		"travel_class":  "1",
		"adults":        "3",
		"children":      "1",
	}

	search := g.NewGoogleSearch(parameter, os.Getenv("API_KEY"))
	results, err := search.GetJSON()
	if err != nil {
		slog.Error(err.Error())
		return err
	}

	// save results to a file if not running in AWS Lambda
	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") == "" {
		file, err := os.Create(fmt.Sprintf("results-%s.json", time.Now().Format("2006-01-02-15-04-05")))
		if err != nil {
			panic(err.Error())
		}
		defer file.Close()

		_, err = file.WriteString(toString(results))
		if err != nil {
			panic(err.Error())
		}
	}

	otherFlightsIntf := results["other_flights"]
	otherFlights, ok := otherFlightsIntf.([]any)
	if !ok {
		slog.Error("other_flights is not an array: %v", otherFlightsIntf)
		return fmt.Errorf("other_flights is not an array")
	}
	otherFlightsBytes, err := json.Marshal(otherFlights)
	if err != nil {
		slog.Error(fmt.Sprintf("error marshalling other_flights: %s", err.Error()))
		return err
	}

	var flights Flights
	err = json.Unmarshal(otherFlightsBytes, &flights)
	if err != nil {
		slog.Error(fmt.Sprintf("error unmarshalling other_flights: %s", err.Error()))
		return err
	}

	flights = flights.Filter()

	slog.Info(fmt.Sprintf("found %d flights", len(flights)))

	for _, flight := range flights {
		slog.Info(fmt.Sprintf("flight: %s", toString(flight)))
		err = repo.SaveFlight(ctx, &flight)
		if err != nil {
			slog.Error(fmt.Sprintf("error saving flight: %s", err.Error()))
			return err
		}
	}

	// save lowest
	lowestPrice, err := repo.GetLowestPrice(ctx)
	if err != nil {
		slog.Error(fmt.Sprintf("error getting lowest price: %s", err.Error()))
		return err
	}
	slog.Info(fmt.Sprintf("lowest price: %s", toString(lowestPrice)))

	currentLowestPrice := flights.GetLowestPrice()
	if lowestPrice == nil || currentLowestPrice.Price < lowestPrice.Price {
		slog.Info(fmt.Sprintf("new lowest price: %s", toString(currentLowestPrice)))
		err = repo.SaveLowestPrice(ctx, currentLowestPrice)
		if err != nil {
			slog.Error(fmt.Sprintf("error saving lowest price: %s", err.Error()))
			return err
		}

		err = mail.SendMail(currentLowestPrice)
		if err != nil {
			slog.Error(fmt.Sprintf("error sending mail: %s", err.Error()))
			return err
		}
	}
	return nil
}

func toString(i any) string {
	s, err := json.Marshal(i)
	if err != nil {
		return fmt.Sprintf("%v", i)
	}
	return string(s)
}

func main() {
	err := initDependencies()
	if err != nil {
		panic(err.Error())
	}

	if os.Getenv("DEBUG") != "" {
		slog.SetDefault(slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
				Level: slog.LevelDebug,
			}),
		))
	}

	if os.Getenv("AWS_LAMBDA_FUNCTION_NAME") != "" {
		lambda.Start(handler)
	} else {
		handler(context.Background())
	}
}
