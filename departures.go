package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/buengese/departures/api"

	w "github.com/buengese/departures/widgets"
	termui "github.com/gizak/termui/v3"
)

var (
	drawInterval = time.Second

	grid *termui.Grid
	stat *w.StationWidget

	colorFg          = 7
	colorBg          = -1
	colorBorderLabel = 7
	colorBorderLine  = 6
)

func main() {
	var (
		stationName string
		search      string
		configFile  string
	)

	settings := w.StationSettings{}

	flag.StringVar(&configFile, "config", "", "Config file")
	flag.StringVar(&settings.ID, "id", "", "ID of the stop")
	flag.StringVar(&settings.FilterMode, "filter-mode", "", "Filter the list for this mode of transporation (Comma separated)")
	flag.StringVar(&settings.FilterDestination, "filter-destination", "", "Filter the list for this destination (Comma separated)")
	flag.StringVar(&settings.FilterLine, "filter-line", "", "Filter the list for this line (Comma separated)")
	flag.IntVar(&settings.Min, "min", 60, "Number of minutes you want to see the departures for")
	flag.StringVar(&search, "search", "", "Search for the stop name to get the stop ID")
	flag.StringVar(&stationName, "station", "", "Fetch departures for given station. Ignored if ID is provided")
	flag.BoolVar(&settings.Bicycle, "bicycle", false, "Only show connections that allow bicycle conveyance.")
	flag.Parse()

	// check if the user just wants to find the station ID we'll exit afterwards
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

	// if the user supplies a config we'll use it
	var config *w.Config
	if configFile != "" {
		config = loadConfig(configFile)
	} else {
		if settings.ID == "" {
			settings.ID = "900000051353"
			settings.Name = "A7"
		}
		config = &w.Config{
			UpdateInterval: time.Minute,
			Stations:       []w.StationSettings{settings},
		}
	}

	if err := termui.Init(); err != nil {
		fmt.Printf("failed to initialize termui: %v", err)
		os.Exit(1)
	}
	defer termui.Close()

	setupStyles()
	stat = w.NewStationWidget(config)

	grid = termui.NewGrid()
	grid.Set(termui.NewRow(1.0, stat))

	termWidth, termHeight := termui.TerminalDimensions()
	grid.SetRect(0, 0, termWidth, termHeight)

	termui.Render(grid)
	eventLoop()
}

func loadConfig(configName string) *w.Config {
	var config *w.Config
	configFile, err := os.Open(configName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config File: %s\n", err.Error())
		os.Exit(1)
	}
	defer configFile.Close()

	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)

	return config
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
