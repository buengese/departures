package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

func getJSON(v interface{}, urlFormat string, values ...interface{}) error {
	resp, err := http.Get(fmt.Sprintf(urlFormat, values...))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	return d.Decode(v)
}

type Station struct {
	Type     string `json:"type"`
	ID       string `json:"id"`
	Name     string `json:"name"`
	Location struct {
		Type      string  `json:"type"`
		ID        string  `json:"id"`
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Products struct {
		Suburban bool `json:"suburban"`
		Subway   bool `json:"subway"`
		Tram     bool `json:"tram"`
		Bus      bool `json:"bus"`
		Ferry    bool `json:"ferry"`
		Express  bool `json:"express"`
		Regional bool `json:"regional"`
	} `json:"products"`
}

func SearchStations(name string) ([]Station, error) {
	var stations []Station
	err := getJSON(&stations, "https://2.bvg.transport.rest/locations?query=%s&poi=false&addresses=false", name)

	return stations, err
}

func GetDepartures(id string, min int) ([]Result, error) {
	var deps []Result
	err := getJSON(&deps, "https://2.bvg.transport.rest/stations/%s/departures?duration=%d", id, min)
	return deps, err
}

type Result struct {
	TripID string `json:"tripId"`
	Stop   struct {
		Type     string `json:"type"`
		ID       string `json:"id"`
		Name     string `json:"name"`
		Location struct {
			Type      string  `json:"type"`
			ID        string  `json:"id"`
			Latitude  float64 `json:"latitude"`
			Longitude float64 `json:"longitude"`
		} `json:"location"`
		Products struct {
			Suburban bool `json:"suburban"`
			Subway   bool `json:"subway"`
			Tram     bool `json:"tram"`
			Bus      bool `json:"bus"`
			Ferry    bool `json:"ferry"`
			Express  bool `json:"express"`
			Regional bool `json:"regional"`
		} `json:"products"`
	} `json:"stop"`
	When      time.Time `json:"when"`
	Direction string    `json:"direction"`
	Line      struct {
		Type     string `json:"type"`
		ID       string `json:"id"`
		FahrtNr  string `json:"fahrtNr"`
		Name     string `json:"name"`
		Public   bool   `json:"public"`
		Mode     string `json:"mode"`
		Product  string `json:"product"`
		Operator struct {
			Type string `json:"type"`
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"operator"`
		Symbol  string `json:"symbol"`
		Nr      int    `json:"nr"`
		Metro   bool   `json:"metro"`
		Express bool   `json:"express"`
		Night   bool   `json:"night"`
	} `json:"line"`
	Remarks []struct {
		Type string `json:"type"`
		Code string `json:"code"`
		Text string `json:"text"`
	} `json:"remarks"`
	Delay    int    `json:"delay"`
	Platform string `json:"platform"`
}
