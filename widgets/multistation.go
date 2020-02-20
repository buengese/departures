package widgets

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/buengese/departures/api"
	"github.com/buengese/departures/ui"

	termui "github.com/gizak/termui/v3"
)

var (
	styleEarly  = termui.NewStyle(3)
	styleOnTime = termui.NewStyle(10)
	styleLate   = termui.NewStyle(1)
)

// Config blabla
type Config struct {
	UpdateInterval time.Duration     `json:"updateinterval"`
	Stations       []StationSettings `json:"stations"`
}

// StationSettings contains general configuration for each monitored station
// e.g ID and filtering settings
type StationSettings struct {
	ID                string `json:"id"`
	Name              string `json:"name"`
	FilterMode        string `json:"filtername"`
	FilterDestination string `json:"filterdestination"`
	FilterLine        string `json:"filterline"`
	Min               int    `json:"min"`
	Bicycle           bool   `json:"bicycle"`
}

// StationWidget represents a Station and display's it in table form
type StationWidget struct {
	*ui.Table
	settings          []StationSettings
	departures        []api.Result
	stationDepartures [][]api.Result
	updateInterval    time.Duration
}

// NewStationWidget constructs a new StationWidget with the given settings
func NewStationWidget(config *Config) *StationWidget {
	self := &StationWidget{
		Table:             ui.NewTable(),
		settings:          config.Stations,
		updateInterval:    config.UpdateInterval,
		stationDepartures: make([][]api.Result, len(config.Stations)),
	}
	self.Title = " Station "
	self.Footer = " Last refresh: never "
	self.ColGap = 3
	self.PadLeft = 2

	if len(config.Stations) > 1 {
		self.Header = []string{"Line", "Station", "Destination", "Time"}
		self.ColResizer = func() {
			self.ColWidths = []int{
				4, 6, maxInt(self.Inner.Dx()-32, 10), 10,
			}
		}
	} else {
		self.Header = []string{"Line", "Destination", "Time"}
		self.ColResizer = func() {
			self.ColWidths = []int{
				4, maxInt(self.Inner.Dx()-26, 10), 10,
			}
		}
	}

	for i := range self.settings {
		self.updateStation(i)
	}

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
	for i := range station.settings {
		station.updateStation(i)
	}

	var departures []api.Result
	for _, deps := range station.stationDepartures {
		departures = append(departures, deps...)
	}

	sort.Slice(departures, func(i, j int) bool {
		return departures[i].When.Before(departures[j].When)
	})
	station.departures = departures

	station.generateTable()
	station.Footer = fmt.Sprintf(" Last refresh: %s ", time.Now().Format("15:04:05"))
}

func (station *StationWidget) updateStation(i int) {
	deps, err := api.GetDepartures(station.settings[i].ID, station.settings[i].Min)
	if err != nil {
		return
	}

	// initialize the filters
	fm := filterSlice(station.settings[i].FilterMode)
	fd := filterSlice(station.settings[i].FilterDestination)
	fl := filterSlice(station.settings[i].FilterLine)

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
		dep.Station = station.settings[i].Name

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
		if station.settings[i].Bicycle && filterBike(dep) {
			continue
		}

		// the entry survived the filters, append it to the filtered list
		filteredDeps = append(filteredDeps, dep)
	}
	station.stationDepartures[i] = filteredDeps
}

func (station *StationWidget) generateTable() {
	strings := make([][]string, len(station.departures))
	styles := make([][]*termui.Style, len(station.departures))

	for i := range station.departures {
		strings[i] = make([]string, 4)
		styles[i] = make([]*termui.Style, 4)

		k := 0
		strings[i][k] = station.departures[i].Line.Name
		k++
		if len(station.settings) > 1 {
			strings[i][k] = station.departures[i].Station
			k++
		}
		strings[i][k] = station.departures[i].Direction
		k++
		strings[i][k] = departureTime(station.departures[i])

		styles[i][k] = &styleOnTime
		if station.departures[i].Delay > 0 {
			styles[i][k] = &styleEarly
		}
		if station.departures[i].Delay < 0 {
			styles[i][k] = &styleLate
		}
	}

	station.Rows = strings
	station.Styles = styles
}

// -------------------------------------------------------------------------

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

func filterBike(r api.Result) bool {
	for _, rem := range r.Remarks {
		if strings.TrimSpace(rem.Code) == "FB" {
			return false
		}
	}
	return true
}

func departureTime(r api.Result) string {
	if r.Delay == 0 {
		return r.When.Format("15:04")
	}
	return fmt.Sprintf("%s (%+d)", r.When.Format("15:04"), r.Delay/60)
}

// -------------------------------------------------------------------------
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
