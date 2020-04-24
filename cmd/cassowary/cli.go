package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/fatih/color"
	"github.com/rogerwelin/cassowary/pkg/client"
	"github.com/urfave/cli"
)

var (
	version             = "dev"
	errConcurrencyLevel = errors.New("Error: Concurrency level cannot be set to: 0")
	errRequestNo        = errors.New("Error: No. of request cannot be set to: 0")
	errNotValidURL      = errors.New("Error: Not a valid URL. Must have the following format: http{s}://{host}")
	errNotValidHeader   = errors.New("Error: Not a valid header value. Did you forget : ?")
	errDurationValue    = errors.New("Error: Duration cannot be set to 0 or negative")
)

func outPutResults(metrics client.ResultMetrics) {
	printf(summaryTable,
		color.CyanString(fmt.Sprintf("%.2f", metrics.TCPStats.TCPMean)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.TCPStats.TCPMedian)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.TCPStats.TCP95p)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.ProcessingStats.ServerProcessingMean)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.ProcessingStats.ServerProcessingMedian)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.ProcessingStats.ServerProcessing95p)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.ContentStats.ContentTransferMean)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.ContentStats.ContentTransferMedian)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.ContentStats.ContentTransfer95p)),
		color.CyanString(strconv.Itoa(metrics.TotalRequests)),
		color.CyanString(strconv.Itoa(metrics.FailedRequests)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.DNSMedian)),
		color.CyanString(fmt.Sprintf("%.2f", metrics.RequestsPerSecond)),
	)
}

func outPutJSON(fileName string, metrics client.ResultMetrics) error {
	if fileName == "" {
		// default filename for json metrics output.
		fileName = "out.json"
	}
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	return enc.Encode(metrics)
}

func runLoadTest(c *client.Cassowary) error {
	metrics, err := c.Coordinate()
	if err != nil {
		return err
	}
	outPutResults(metrics)

	if c.ExportMetrics {
		return outPutJSON(c.ExportMetricsFile, metrics)
	}
	return nil
}

func validateCLI(c *cli.Context) error {

	prometheusEnabled := false
	var header []string
	var httpMethod string
	var data []byte
	duration := 0

	if c.Int("concurrency") == 0 {
		return errConcurrencyLevel
	}

	if c.Int("requests") == 0 {
		return errRequestNo
	}

	if c.String("duration") != "" {
		var err error
		duration, err = strconv.Atoi(c.String("duration"))
		if err != nil {
			return err
		}
		if duration <= 0 {
			return errDurationValue
		}
	}

	if client.IsValidURL(c.String("url")) == false {
		return errNotValidURL
	}

	if c.String("prompushgwurl") != "" {
		prometheusEnabled = true
	}

	if c.String("header") != "" {
		length := 0
		length, header = client.SplitHeader(c.String("header"))
		if length != 2 {
			return errNotValidHeader
		}
	}

	if c.String("postfile") != "" {
		httpMethod = "POST"
		fileData, err := readFile(c.String("postfile"))
		if err != nil {
			return err
		}
		data = fileData
	} else if c.String("putfile") != "" {
		httpMethod = "PUT"
		fileData, err := readFile(c.String("putfile"))
		if err != nil {
			return err
		}
		data = fileData
	} else {
		httpMethod = "GET"
	}

	cass := &client.Cassowary{
		FileMode:          false,
		BaseURL:           c.String("url"),
		ConcurrencyLevel:  c.Int("concurrency"),
		Requests:          c.Int("requests"),
		RequestHeader:     header,
		Duration:          duration,
		PromExport:        prometheusEnabled,
		PromURL:           c.String("prompushgwurl"),
		ExportMetrics:     c.Bool("json-metrics"),
		ExportMetricsFile: c.String("json-metrics-file"),
		DisableKeepAlive:  c.Bool("disable-keep-alive"),
		Timeout:           c.Int("timeout"),
		HTTPMethod:        httpMethod,
		Data:              data,
	}

	return runLoadTest(cass)
}

func validateCLIFile(c *cli.Context) error {
	prometheusEnabled := false
	var header []string

	if c.Int("concurrency") == 0 {
		return errConcurrencyLevel
	}

	if client.IsValidURL(c.String("url")) == false {
		return errNotValidURL
	}

	if c.String("prompushgwurl") != "" {
		prometheusEnabled = true
	}

	if c.String("header") != "" {
		length := 0
		length, header = client.SplitHeader(c.String("header"))
		if length != 2 {
			return errNotValidHeader
		}
	}

	urlSuffixes, err := readLocalRemoteFile(c.String("file"))
	if err != nil {
		return nil
	}

	cass := &client.Cassowary{
		FileMode:          true,
		BaseURL:           c.String("url"),
		ConcurrencyLevel:  c.Int("concurrency"),
		RequestHeader:     header,
		PromExport:        prometheusEnabled,
		PromURL:           c.String("prompushgwurl"),
		ExportMetrics:     c.Bool("json-metrics"),
		ExportMetricsFile: c.String("json-metrics-file"),
		DisableKeepAlive:  c.Bool("diable-keep-alive"),
		Timeout:           c.Int("timeout"),
		Requests:          c.Int("requests"),
		URLPaths:          urlSuffixes,
		HTTPMethod:        "GET",
	}

	return runLoadTest(cass)
}

func runCLI(args []string) {
	app := cli.NewApp()
	app.Name = "cassowary - 學名"
	app.HelpName = "cassowary"
	app.UsageText = "cassowary [command] [command options] [arguments...]"
	app.EnableBashCompletion = true
	app.Usage = ""
	app.Version = version
	app.Commands = []cli.Command{
		{
			Name:  "run-file",
			Usage: "start load test in spread mode",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "u, url",
					Usage:    "the base url (absoluteURI) to be used",
					Required: true,
				},
				cli.IntFlag{
					Name:     "c, concurrency",
					Usage:    "number of concurrent users",
					Required: true,
				},
				cli.IntFlag{
					Name:  "n, requests",
					Usage: "number of requests to perform",
				},
				cli.IntFlag{
					Name:  "t, timeout",
					Usage: "http client timeout",
					Value: 5,
				},
				cli.StringFlag{
					Name:     "f, file",
					Usage:    "specify `FILE` path, local or www, containing the url suffixes",
					Required: true,
				},
				cli.StringFlag{
					Name:  "p, prompushgwurl",
					Usage: "specify prometheus push gateway url to send metrics (optional)",
				},
				cli.StringFlag{
					Name:  "H, header",
					Usage: "add arbitrary header, eg. 'Host: www.example.com'",
				},
				cli.BoolFlag{
					Name:  "F, json-metrics",
					Usage: "outputs metrics to a json file by setting flag to true",
				},
				cli.StringFlag{
					Name:  "json-metrics-file",
					Usage: "outputs metrics to a custom json filepath, if json-metrics is set to true",
				},
				cli.BoolFlag{
					Name:  "disable-keep-alive",
					Usage: "use this flag to disable http keep-alive",
				},
			},
			Action: validateCLIFile,
		},
		{
			Name:  "run",
			Usage: "start load-test",
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:     "u, url",
					Usage:    "the url (absoluteURI) to be used",
					Required: true,
				},
				cli.IntFlag{
					Name:     "c, concurrency",
					Usage:    "number of concurrent users",
					Required: true,
				},
				cli.IntFlag{
					Name:     "n, requests",
					Usage:    "number of requests to perform",
					Required: true,
				},
				cli.StringFlag{
					Name:  "d, duration",
					Usage: "set the duration in seconds of the load test (example: do 100 requests in a duration of 30s)",
				},
				cli.IntFlag{
					Name:  "t, timeout",
					Usage: "http client timeout",
					Value: 5,
				},
				cli.StringFlag{
					Name:  "p, prompushgwurl",
					Usage: "specify prometheus push gateway url to send metrics (optional)",
				},
				cli.StringFlag{
					Name:  "H, header",
					Usage: "add arbitrary header, eg. 'Host: www.example.com'",
				},
				cli.BoolFlag{
					Name:  "F, json-metrics",
					Usage: "outputs metrics to a json file by setting flag to true",
				},
				cli.StringFlag{
					Name:  "postfile",
					Usage: "file containing data to POST (content type will default to application/json)",
				},
				cli.StringFlag{
					Name:  "putfile",
					Usage: "file containig data to PUT (content type will default to application/json)",
				},
				cli.StringFlag{
					Name:  "json-metrics-file",
					Usage: "outputs metrics to a custom json filepath, if json-metrics is set to true",
				},
				cli.BoolFlag{
					Name:  "disable-keep-alive",
					Usage: "use this flag to disable http keep-alive",
				},
			},
			Action: validateCLI,
		},
	}

	err := app.Run(args)
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
}
