package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/buengese/departures/api"

	w "github.com/buengese/departures/widgets"
	termui "github.com/gizak/termui/v3"
	"gopkg.in/AlecAivazis/survey.v1"
)

var (
	drawInterval = time.Second

	grid *termui.Grid
	stat *w.StationWidget

	// TODO make this configurable?
	colorFg          = 7
	colorBg          = -1
	colorBorderLabel = 7
	colorBorderLine  = 6
)

type flagArray []string

func (i *flagArray) String() string {
	return strings.Join(*i, ",")
}

func (i *flagArray) Set(value string) error {
	*i = append(*i, value)
	return nil
}

func main() {
	// parse the command line arguments
	settings := &w.StationSettings{}

	var (
		stationName string
		search      string
		ids         flagArray
	)

	flag.Var(&ids, "id", "ID of the stop")
	flag.StringVar(&settings.FilterMode, "filter-mode", "", "Filter the list for this mode of transporation (Comma separated)")
	flag.StringVar(&settings.FilterDestination, "filter-destination", "", "Filter the list for this destination (Comma separated)")
	flag.StringVar(&settings.FilterLine, "filter-line", "", "Filter the list for this line (Comma separated)")
	flag.IntVar(&settings.Min, "min", 60, "Number of minutes you want to see the departures for")
	flag.StringVar(&search, "search", "", "Search for the stop name to get the stop ID")
	flag.StringVar(&stationName, "station", "", "Fetch departures for given station. Ignored if ID is provided")
	flag.BoolVar(&settings.Bicycle, "bicycle", false, "Only show connections that allow bicycle conveyance.")
	flag.Parse()

	// check if the user just wants to find the station ID
	if search != "" {
		stations, err := api.SearchStations(search)
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

	if len(ids) == 0 {
		settings.IDs = []string{"900000051353", "900000051303"}
	} else {
		settings.IDs = ids
	}

	if err := termui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
	}
	defer termui.Close()

	setupStyles()
	stat = w.NewStationWidget(settings)

	// using a grid here would allow to display multiple station widgets at
	// once tmux style. this hasn't been implemented yet
	grid = termui.NewGrid()
	grid.Set(termui.NewRow(1.0, stat))
	termWidth, termHeight := termui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	termui.Render(grid)
	eventLoop()
}

func setupStyles() {
	termui.Theme.Default = termui.NewStyle(termui.Color(colorFg), termui.Color(colorBg))
	termui.Theme.Block.Title = termui.NewStyle(termui.Color(colorBorderLabel), termui.Color(colorBg))
	termui.Theme.Block.Border = termui.NewStyle(termui.Color(colorBorderLine), termui.Color(colorBg))
}

func eventLoop() {
	drawTicker := time.NewTicker(drawInterval).C

	sigTerm := make(chan os.Signal, 2)
	signal.Notify(sigTerm, os.Interrupt, syscall.SIGTERM)

	uiEvents := termui.PollEvents()

	for {
		select {
		case <-sigTerm:
			return
		case <-drawTicker:
			termui.Render(grid)
		case e := <-uiEvents:
			switch e.ID {
			case "q", "<C-c>":
				return
			case "<Resize>":
				payload := e.Payload.(termui.Resize)
				grid.SetRect(0, 0, payload.Width, payload.Height)
				termui.Clear()
			case "k", "<Up>", "<MouseWheelUp>":
				stat.ScrollUp()
				termui.Render(stat)
			case "j", "<Down>", "<MouseWheelDown>":
				stat.ScrollDown()
				termui.Render(stat)
			}
		}
	}
}

// -------------------------------------------------------------------------------

func promptForStation(name string) (*api.Station, error) {
	stations, err := api.SearchStations(name)
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
	optionStation := map[string]api.Station{}
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
