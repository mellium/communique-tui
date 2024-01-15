package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"
)

var (
	useQuic bool
	isC2C   bool
)

func main() {
	flags := flag.NewFlagSet("XMPP Benchmark", flag.ContinueOnError)
	flags.BoolVar(&isC2C, "c2c", isC2C, "Start test for C2C (C2S otherwise)")
	flags.BoolVar(&useQuic, "quic", useQuic, "Start session over quic (XEP-0467)")
	flags.SetOutput(ioutil.Discard)
	err := flags.Parse(os.Args[1:])
	if err != nil {
		os.Exit(2)
	}
	pref := "c2s-"
	if isC2C {
		pref = "c2c-"
	}
	reportName := pref + "report-tcp.csv"
	if useQuic {
		fmt.Println("Using QUIC")
		reportName = pref + "report-quic.csv"
	}

	file, err := os.OpenFile(reportName, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		panic(err)
	}

	if _, err := file.WriteString("timestamp, connectionCount, averageLatency\n"); err != nil {
		panic(err)
	}

	defer closeClient()

	for clientCount < 500 {
		startMultiConn(50)
		fmt.Printf("Created %d clients\n", clientCount)
		time.Sleep(2 * time.Second)
		curTime := time.Now()
		var averageTime float64
		if isC2C {
			averageTime = c2cBatchTest()
		} else {
			averageTime = c2sBatchTest()
		}
		fmt.Printf("Average time for %d clients is %f\n", clientCount, averageTime)
		if _, err := file.WriteString(fmt.Sprintf("%s, %d, %f\n", curTime.Format(time.RFC3339), clientCount, averageTime)); err != nil {
			panic(err)
		}
		time.Sleep(3 * time.Second)
	}
}
