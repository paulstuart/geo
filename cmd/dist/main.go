package main

import (
	"flag"
	"fmt"
	"log"
	_ "net/http/pprof"
	"os"

	"github.com/paulstuart/geo"
)

var (
	miles bool
)

func main() {
	flag.BoolVar(&miles, "miles", false, "calculate distance in miles (vs km)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("usage: %s <pt1> <pt2>", os.Args[0])
	}

	src := args[0]
	loc := args[1]

	pt1, err := geo.QueryCoords(src)
	if err != nil {
		log.Fatal(err)
	}

	pt2, err := geo.QueryCoords(loc)
	if err != nil {
		log.Fatal(err)
	}

	units := "km"
	dist := geo.Distance(pt1[0], pt1[1], pt2[0], pt2[1])
	if miles {
		dist = dist / geo.MilesToKilometer
		units = "mi"
	}
	fmt.Printf("%.2f %s\n", dist, units)
}
