package main

import (
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/smithy-go/ptr"
)

type Mail struct {
	conn *ses.SES
	from string
	to   []string
}

func NewMail(conn *ses.SES) *Mail {
	from := os.Getenv("MAIL_FROM")
	if from == "" {
		panic("MAIL_FROM is required")
	}
	to_list := os.Getenv("MAIL_TO_LIST")
	if to_list == "" {
		panic("MAIL_TO_LIST is required")
	}
	to := strings.Split(to_list, ",")
	return &Mail{conn: conn, from: from, to: to}
}

func (m *Mail) SendMail(filters map[string]string, input Input, fligh *Flight) error {
	newFlightParagraph := fmt.Sprintf(`
		<p> %s</p> <br>
		<p> Pessoas: %s adultos e %s criancas </p> <br>
		<p> Saida %s de %s </p> <br>
		<p> Chegada %s de %s </p> <br>
		<p> Duracao: %s h </p> <br>
		<p> R$ %s</p> <br>
		<p> link: %s</p> <br>
		`,
		fligh.AirlineFlights[0].Airline,
		filters["adults"],
		filters["children"],
		fligh.AirlineFlights[0].DepartueAirport.Time,
		fligh.AirlineFlights[0].DepartueAirport.Name,
		fligh.AirlineFlights[0].ArrivalAirport.Time,
		fligh.AirlineFlights[0].ArrivalAirport.Name,
		strconv.Itoa(fligh.AirlineFlights[0].Duration/60),
		strconv.Itoa(fligh.Price),
		fligh.AirlineFlights[0].DepartueAirport.ID,
	)

	for _, to := range m.to {
		_, err := m.conn.SendEmail(&ses.SendEmailInput{
			Destination: &ses.Destination{
				ToAddresses: []*string{
					ptr.String(to),
				},
			},
			Source: ptr.String(m.from),
			Message: &ses.Message{
				Subject: &ses.Content{
					Data: ptr.String("Flight Price Alert"),
				},
				Body: &ses.Body{
					Html: &ses.Content{
						Data: ptr.String("<h1>Alerta de Preco</h1><p>Voo com preco abaixo do registrado encontrado</p>" + newFlightParagraph),
					},
				},
			},
		})
		if err != nil {
			emailNotVerifiedError := strings.Contains(err.Error(), "Email address is not verified")
			if emailNotVerifiedError {
				m.conn.VerifyEmailIdentityRequest(&ses.VerifyEmailIdentityInput{
					EmailAddress: ptr.String(to),
				})

				continue
			}
			return err
		}

		slog.Info(fmt.Sprintf("email sent to: %s", to))
	}

	slog.Info(fmt.Sprintf("sent to all recipients: %s", newFlightParagraph))
	return nil
}

func (m *Mail) SendErrorMail(err error) error {
	slog.Error(fmt.Sprintf("error: %s", err.Error()))

	for _, to := range m.to {
		_, err := m.conn.SendEmail(&ses.SendEmailInput{
			Destination: &ses.Destination{
				ToAddresses: []*string{
					ptr.String(to),
				},
			},
			Source: ptr.String(m.from),
			Message: &ses.Message{
				Subject: &ses.Content{
					Data: ptr.String("Flight Price Alert Service - Error"),
				},
				Body: &ses.Body{
					Text: &ses.Content{
						Data: ptr.String("found and error in flight price alert service"),
					},
				},
			},
		})
		if err != nil {
			emailNotVerifiedError := strings.Contains(err.Error(), "Email address is not verified")
			if emailNotVerifiedError {
				m.conn.VerifyEmailIdentityRequest(&ses.VerifyEmailIdentityInput{
					EmailAddress: ptr.String(to),
				})

				continue
			}
			return err
		}
	}
	return nil
}
