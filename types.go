package main

type Flights []Flight

type Flight struct {
	AirlineLogo    string          `json:"airline_logo"`
	DepartueToken  string          `json:"departure_token"`
	AirlineFlights []AirlineFlight `json:"flights"`
	Price          int             `json:"price"`
	Type           string          `json:"type"`
}

type AirlineFlight struct {
	Airline         string  `json:"airline"`
	AirlineLogo     string  `json:"airline_logo"`
	ArrivalAirport  Airport `json:"arrival_airport"`
	DepartueAirport Airport `json:"departure_airport"`
	Duration        int     `json:"duration"`
}

type Airport struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Time string `json:"time"`
}

func (f Flights) GetLowestPrice() *Flight {
	if len(f) == 0 {
		return nil
	}

	lowestPrice := f[0]
	for _, flight := range f {
		if flight.Price < lowestPrice.Price {
			lowestPrice = flight
		}
	}

	return &lowestPrice
}

func (f Flights) Filter() Flights {
	var filtered Flights
	for _, flight := range f {
		if flight.Price > 0 {
			filtered = append(filtered, flight)
		}
	}
	return filtered
}
