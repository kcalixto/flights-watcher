package main

import (
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

func (m *Mail) SendMail(fligh *Flight) error {
	newFlightParagraph := `
		<p> ` + fligh.AirlineFlights[0].Airline + `</p> <br>
		<p> Duracao: ` + strconv.Itoa(fligh.AirlineFlights[0].Duration/60) + `h </p> <br>
		<p> R$ ` + strconv.Itoa(fligh.Price) + `</p> <br>
		<p> link: ` + fligh.AirlineFlights[0].DepartueAirport.ID + `</p> <br>
	`

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
	}
	return nil
}
