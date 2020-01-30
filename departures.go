package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	ui "github.com/gizak/termui/v3"
	w "github.com/noxer/departures/widgets"
	"gopkg.in/AlecAivazis/survey.v1"
)

var (
	drawInterval = time.Second

	grid *ui.Grid
	stat *w.StationWidget

	colorFg          = 7
	colorBg          = -1
	colorBorderLabel = 7
	colorBorderLine  = 6
)

func setDefaultTermuiColors() {
	ui.Theme.Default = ui.NewStyle(ui.Color(colorFg), ui.Color(colorBg))
	ui.Theme.Block.Title = ui.NewStyle(ui.Color(colorBorderLabel), ui.Color(colorBg))
	ui.Theme.Block.Border = ui.NewStyle(ui.Color(colorBorderLine), ui.Color(colorBg))
}

func setupGrid() {
	grid = ui.NewGrid()
	grid.Set(ui.NewRow(1.0, stat))
}

func eventLoop() {
	drawTicker := time.NewTicker(drawInterval).C

	sigTerm := make(chan os.Signal, 2)
	signal.Notify(sigTerm, os.Interrupt, syscall.SIGTERM)

	uiEvents := ui.PollEvents()

	for {
		select {
		case <-sigTerm:
			return
		case <-drawTicker:
			ui.Render(grid)
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(ui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				ui.Clear()
			case "k", "<Up>", "<MouseWheelUp>":
				stat.ScrollUp()
				ui.Render(stat)
			case "j", "<Down>", "<MouseWheelDown>":
				stat.ScrollDown()
				ui.Render(stat)
			}
		}
	}
}

func main() {
	// parse the command line arguments
	settings := &w.StationSettings{}

	flag.StringVar(&settings.ID, "id", "", "ID of the stop")
	flag.StringVar(&settings.FilterMode, "filter-mode", "", "Filter the list for this mode of transporation (Comma separated)")
	flag.StringVar(&settings.FilterDestination, "filter-destination", "", "Filter the list for this destination (Comma separated)")
	flag.StringVar(&settings.FilterLine, "filter-line", "", "Filter the list for this line (Comma separated)")
	flag.IntVar(&settings.Width, "width", intEnv("WTF_WIDGET_WIDTH"), "Width of the output")
	flag.IntVar(&settings.Retries, "retries", 3, "Number of retries before giving up")
	flag.DurationVar(&settings.RetryPause, "retry-pause", time.Second, "Pause between retries")
	flag.IntVar(&settings.Min, "min", 60, "Number of minutes you want to see the departures for")
	flag.BoolVar(&settings.ForceColor, "force-color", false, "Use this flag to enforce color output even if the terminal does not report support")
	flag.StringVar(&settings.Search, "search", "", "Search for the stop name to get the stop ID")
	flag.StringVar(&settings.StationName, "station", "", "Fetch departures for given station. Ignored if ID is provided")
	flag.BoolVar(&settings.Verbose, "verbose", false, "Be verbose and show additional information (mode of transportataion, operator and additional remarks).")
	flag.BoolVar(&settings.Bicycle, "bicycle", false, "Only show connections that allow bicycle conveyance.")
	flag.Parse()

	// check if the user just wants to find the station ID
	if settings.Search != "" {
		stations, err := searchStations(settings.Search)
		if err != nil {
			fmt.Println("Could not query stations")
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}

		fmt.Printf("Found %d station(s):\n", len(stations))
		for _, s := range stations {
			fmt.Printf("  %s - %s\n", s.ID, s.Name)
		}

		return
	}

	// search of the station and provide user option to choose
	if settings.ID == "" && settings.StationName != "" {
		s, err := promptForStation(settings.StationName)
		if err != nil {
			fmt.Println(err)
		} else {
			settings.ID = s.ID
		}
	}

	// set default id if empty
	if settings.ID == "" {
		//fmt.Println("station ID is empty. Defaulting to: 900000100003")
		settings.ID = "900000100003"
	}

	if err := ui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	setDefaultTermuiColors()
	stat = w.NewStationWidget(settings)
	setupGrid()

	termWidth, termHeight := ui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	ui.Render(grid)
	eventLoop()
}

func searchStations(name string) ([]station, error) {
	var stations []station
	err := getJSON(&stations, "https://2.bvg.transport.rest/locations?query=%s&poi=false&addresses=false", name)

	return stations, err
}

func promptForStation(name string) (*station, error) {
	stations, err := searchStations(name)
	if err != nil {
		return nil, fmt.Errorf("could not query stations")
	}

	if l := len(stations); l == 0 {
		// no stations found
		return nil, fmt.Errorf("could not find matching stations")
	} else if l == 1 {
		return &stations[0], nil
	}
	// set first result as fallback
	fallback := stations[0].Name

	// convert to map[string]station to get station after user prompt
	var options []string
	optionStation := map[string]station{}
	for _, s := range stations {
		options = append(options, s.Name)
		optionStation[s.Name] = s
	}

	prompt := &survey.Select{
		Message: "Choose a station:",
		Options: options,
		Default: fallback,
	}

	var choice string
	if err = survey.AskOne(prompt, &choice, nil); err != nil {
		fmt.Println("Failed to get answer on station list. Defaulting to", fallback)
		choice = fallback
	}

	s := optionStation[choice]
	return &s, nil
}

func getJSON(v interface{}, urlFormat string, values ...interface{}) error {
	resp, err := http.Get(fmt.Sprintf(urlFormat, values...))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	d := json.NewDecoder(resp.Body)
	return d.Decode(v)
}

func intEnv(key string) int {
	i, _ := strconv.Atoi(os.Getenv(key))
	return i
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
