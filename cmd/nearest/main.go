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
	latLon bool
)

func main() {
	flag.BoolVar(&latLon, "lat", latLon, "coordinates are <lat,lon> (vs lon,lat)")
	flag.Parse()

	args := flag.Args()
	if len(args) < 2 {
		log.Fatalf("usage: %s <file> <loc>", os.Args[0])
	}

	src := args[0]
	loc := args[1]

	pt, err := geo.QueryPoint(loc)
	if err != nil {
		log.Fatal(err)
	}

	info, err := geo.Nearest(src, pt, latLon)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Index:%d Distance:%f Line:%s\n", info.Index, info.Distance, info.Line)
}
