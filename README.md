# departures-interactive
Show departure times for your Berlin public transport station. This is a fork of the original departures intended to run on info displays or in the background

## install
To install from source you need to have a current Go version installed.

```bash
go get -u github.com/buengese/departures
```
Now you should have the `departures` binary installed in your `$GOPATH/bin` directory. You can call it from there or add the directory to your `$PATH`.

## usage
First you need to find out the ID of your station. To do this run the tool with the `-search` parameter.
```bash
~$ departures -search="Alexanderplatz"
```

This should help you identify the station you want to look at. Now you can request the timetable for Alexanderplatz.

```bash
~$ departures -id="900000100003"
```

You can limit the lines and directions shown. Multiple values must be separated by a comma.

```bash
~$ departures -id="900000100003" -filter-line="M4" -filter-destination="S Hackescher Markt"
```

You can also create a config file that allows displaying an arbitrary number of stations at the same time.
An example config file is included in this repository

```bash
~$ departures -config conf.json
```

## attribution
I'm using https://2.bvg.transport.rest to request the current timetable data. Thanks to [derhuerst](https://github.com/derhuerst).
