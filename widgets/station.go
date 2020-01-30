package widgets

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	termui "github.com/gizak/termui/v3"
	"github.com/noxer/departures/ui"
)

var (
	styleEarly  = termui.NewStyle(3)
	styleOnTime = termui.NewStyle(10)
	styleLate   = termui.NewStyle(1)
)

type StationSettings struct {
	ID                string
	FilterMode        string
	FilterDestination string
	FilterLine        string
	Width             int
	Retries           int
	RetryPause        time.Duration
	ForceColor        bool
	Min               int
	Search            string
	StationName       string
	Verbose           bool
	Bicycle           bool
}

type StationWidget struct {
	*ui.Table
	settings       *StationSettings
	departures     []result
	updateInterval time.Duration
}

func NewStationWidget(settings *StationSettings) *StationWidget {
	self := &StationWidget{
		Table:          ui.NewTable(),
		settings:       settings,
		updateInterval: time.Minute,
	}
	self.Title = " Station "
	self.Header = []string{"Line", "Destination", "Time"}
	self.ShowLocation = true
	self.ColGap = 3
	self.PadLeft = 2
	self.ColResizer = func() {
		self.ColWidths = []int{
			4, maxInt(self.Inner.Dx()-26, 10), 10,
		}
	}

	self.update()

	go func() {
		for range time.NewTicker(self.updateInterval).C {
			self.Lock()
			self.update()
			self.Unlock()
		}
	}()

	return self
}

func (station *StationWidget) update() {
	var err error

	// request the departures
	var deps []result
	err = getJSON(&deps, "https://2.bvg.transport.rest/stations/%s/departures?duration=%d", station.settings.ID, station.settings.Min)
	if err != nil {
		//fmt.Println("Could not query departures")
		//fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		return
	}

	// initialize the filters
	fm := filterSlice(station.settings.FilterMode)
	fd := filterSlice(station.settings.FilterDestination)
	fl := filterSlice(station.settings.FilterLine)

	// calculate the length of the columns
	from := time.Now().Add(-2 * time.Minute)
	until := time.Now().Add(time.Hour)
	filteredDeps := deps[:0] // no need to waste space*/

	for _, dep := range deps {
		if dep.When.Before(from) || dep.When.After(until) {
			continue
		}
		// trim unnecessary whitespace
		dep.Line.Product = strings.TrimSpace(dep.Line.Product)
		dep.Direction = strings.TrimSpace(dep.Direction)
		dep.Line.Name = strings.TrimSpace(dep.Line.Name)

		// apply filters
		if isFiltered(fm, dep.Line.Product) {
			continue
		}
		if isFiltered(fd, dep.Direction) {
			continue
		}
		if isFiltered(fl, dep.Line.Name) {
			continue
		}
		if station.settings.Bicycle && filterBike(dep) {
			continue
		}

		// the entry survived the filters, append it to the filtered list
		filteredDeps = append(filteredDeps, dep)
	}
	station.departures = filteredDeps
	station.generateTable()
}

func (station *StationWidget) generateTable() {
	strings := make([][]string, len(station.departures))
	styles := make([][]*termui.Style, len(station.departures))
	for i := range station.departures {
		strings[i] = make([]string, 3)
		styles[i] = make([]*termui.Style, 3)
		strings[i][0] = station.departures[i].Line.Name
		strings[i][1] = station.departures[i].Direction
		strings[i][2] = departureTime(station.departures[i])

		styles[i][2] = &styleOnTime
		if station.departures[i].Delay > 0 {
			styles[i][2] = &styleEarly
		}
		if station.departures[i].Delay < 0 {
			styles[i][2] = &styleLate
		}
	}
	station.Rows = strings
	station.Styles = styles
}

// -------------------------------------------------------------------------

func getJSON(v interface{}, urlFormat string, values ...interface{}) error {
	resp, err := http.Get(fmt.Sprintf(urlFormat, values...))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	return d.Decode(v)
}

func filterSlice(filter string) []string {
	if filter == "" {
		return nil
	}

	fs := strings.Split(strings.ToUpper(filter), ",")
	for i, f := range fs {
		fs[i] = strings.TrimSpace(f)
	}
	return fs
}

func isFiltered(filter []string, v string) bool {
	if len(filter) == 0 {
		return false
	}

	for _, f := range filter {
		if strings.EqualFold(f, v) {
			return false
		}
	}
	return true
}

func filterBike(r result) bool {
	for _, rem := range r.Remarks {
		if strings.TrimSpace(rem.Code) == "FB" {
			return false
		}
	}
	return true
}

func departureTime(r result) string {
	if r.Delay == 0 {
		return r.When.Format("15:04")
	}
	return fmt.Sprintf("%s (%+d)", r.When.Format("15:04"), r.Delay/60)
}

type station struct {
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

type result struct {
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

// ---------------------------------------
// SERIOUSLY THIS SHOULD BE PART OF THE STANDARD LIBRARY!!!
// I don't care that it's just 6 lines. It's just annoying to either have some kind of
// utils package just for this one function or to have this just floating around
// in some other package.
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
