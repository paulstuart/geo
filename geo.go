package geo

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/exp/constraints"
)

var (
	ErrInvalidCoordinates = errors.New("invalid coordinates")
)

type Float = constraints.Float

type Pair[T Float] [2]T
type Rect[T Float] [2]Pair[T]

var (
	//alameda = Pair{AlaLat, AlaLon}
	//longitudeKilometerPerLatitude [91]float64  // lookup table of longitude to Km per each degree latitude
	lonKmLookup [901]float64 // lookup table of longitude to Km per each degree latitude
)

// GeoType for coordinates with slightly less accuracy
// a float32 has 7 digits of precision, which is within ~11cm
//
// Virtually all geo data is "close enough" using this,
// and for data that heavily comprises these points,
// one can reduce memory footprint by one half (for points)
// type GeoType[T Float] T

// GeoPoints provides abstraction for slices of data with coordinates
type GeoPoints[T Float] interface {
	IndexPoint(int) Point[T]
	Len() int
}

const (
	// DegreeToKilometer is a "constant" for latitude but varies for longitude
	DegreeToKilometer     = 111.111 //111.321
	MilesToKilometer      = 1.609344
	EarthRadiusInKM       = 6371.1 // 6378.1
	SquareKmPerSquareMile = 0.386102
)

func init() {
	for i := 0; i < len(lonKmLookup); i++ {
		lat := float64(i) / 10.0
		lonKmLookup[i] = LonKilos(lat)
	}
}

const Radian = math.Pi / 180.0

func deg2rad(d float64) float64 {
	return d * Radian
}

// LookupLonKmPerLat returns the ratio of kilometers to degrees longitude
// at the given latitude.
//
// Accuracy is with 1% under 80 degrees, which is good enough for most work
func LookupLonKmPerLat[T Float](lat T) T {
	idx := int(lat * 10)
	return T(lonKmLookup[idx])
}

// LonKilos returns the kilometers per degree longitude at the given latitude
func LonKilos(lat float64) float64 {
	return math.Cos(deg2rad(lat)) * DegreeToKilometer
}

// LongitudeKilometerDegrees converts the distance given
// in kilometers at that latitude to degrees
func LongitudeKilometerDegrees[T Float](lat T, kilometers T) T {
	return T(kilometers / T(LonKilos(float64(lat))))
}

// LongitudeKilometers returns the distance of the degrees lon, at latitude lat
func LongitudeKilometers(lat, lon float64) float64 {
	return LonKilos(lat) * lon
}

// Expand returns the a box with a radius in kM for
func Expand[T Float](lat, lon T, radiusKM T) Rect[T] {
	latx := radiusKM / lon
	lonx := LongitudeKilometerDegrees[T](lat, radiusKM)
	return Rect[T]{
		Pair[T]{lat - latx, lon - lonx},
		Pair[T]{lat + latx, lon + lonx},
	}
}

// Distance returns the distance in kM between 2 geographic points
// It uses the Haversine formula for spherical calculations
func Distance[T Float](lat1, lon1, lat2, lon2 T) T {
	dlat1 := deg2rad(float64(lat1))
	dlon1 := deg2rad(float64(lon1))
	dlat2 := deg2rad(float64(lat2))
	dlon2 := deg2rad(float64(lon2))

	v := math.Acos(math.Sin(dlat1)*math.Sin(dlat2)+math.Cos(dlat1)*math.Cos(dlat2)*math.Cos(dlon2-dlon1)) * EarthRadiusInKM
	return T(v)
}

func Distance32(lat1, lon1, lat2, lon2 float32) float64 {
	return Distance(float64(lat1), float64(lon1), float64(lat2), float64(lon2))
}

func DistanceGeoType[T Float](lat1, lon1, lat2, lon2 T) T {
	return Distance[T](lat1, lon1, lat2, lon2)
}

// ApproximateDistanceGeo returns the approximate distance between 2 points
// It uses the pythagarean distance calc which is meant for 2d operations
// but is "good enough" for shorter distances (which we primarily care about)
// It is about 7 times faster than the "proper way"
func ApproximateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	//	lonDegreeKm := LookupLonKmPerLatInt(round(float64(lat2)))
	//lonDegreeKm := LookupLonKmPerLatInt(int(lat2))
	lonDegreeKm := LookupLonKmPerLat(lat2)
	a := float64(lat2-lat1) * DegreeToKilometer
	b := float64(lon2-lon1) * lonDegreeKm
	return math.Sqrt(math.Pow(a, 2) + math.Pow(b, 2))
}

// ApproximateDistanceGeo returns the approximate distance between 2 points
// It uses the pythagarean distance calc which is meant for 2d operations
// but is "good enough" for shorter distances (which we primarily care about)
// It is about 7 times faster than the "proper way"
func ApproximateDistanceGeo[T Float](lat1, lon1, lat2, lon2 T) T {
	//	lonDegreeKm := LookupLonKmPerLatInt(round(float64(lat2)))
	lonDegreeKm := LookupLonKmPerLat(float64(lat1)) //LookupLonKmPerLatInt(int(lat2))
	a := float64(lat2-lat1) * DegreeToKilometer
	b := float64(lon2-lon1) * lonDegreeKm
	return T(math.Sqrt(math.Pow(a, 2) + math.Pow(b, 2)))
}

func GeoPoint[T Float](lat, lon float64) Point[T] {
	// return Point[T]{GeoType(lat), GeoType(lon)}
	return Point[T]{T(lat), T(lon)}
}

func AreaInKm(lat1, lon1, lat2, lon2 float64) float64 {
	delta := math.Abs(lon2 - lon1)
	d1 := LongitudeKilometers(lat1, delta)
	d2 := LongitudeKilometers(lat2, delta)
	avg := math.Abs(d2+d1) / 2
	h := math.Abs(lat2-lat1) * DegreeToKilometer
	a := h * avg
	return a
}

func SquareKmInMiles(k float64) float64 {
	return k * SquareKmPerSquareMile
}

func AreaInMiles(lat1, lon1, lat2, lon2 float64) float64 {
	return SquareKmInMiles(AreaInKm(lat1, lon1, lat2, lon2))
}

func Coords[T Float](query string) ([]T, error) {
	parts := strings.Split(query, ",")
	if len(parts) != 4 {
		parts = strings.Split(query, "/")
		if len(parts) != 4 {
			return nil, ErrInvalidCoordinates
		}
	}
	coords, err := geos[T](parts...)
	if err != nil {
		return nil, fmt.Errorf("parse failure (%v): %w", err, ErrInvalidCoordinates)
	}
	if err := coordCheck(coords...); err != nil {
		for i := len(coords); i < 4; i++ {
			coords = append(coords, 0)
		}
		return nil, fmt.Errorf(badPrefix+" %w", err)
	}
	return coords, nil
}

type Point[T Float] struct {
	Lat, Lon T
}

// Less returns true if it is less than the given point
func (p Point[T]) Less(x Point[T]) bool {
	if p.Lat < x.Lat {
		return true
	} else if p.Lat > x.Lat {
		return false
	} else {
		// lon is secondary sort
		return p.Lon < x.Lon
	}
}

// Label returns a consistent string representation of the coordinates
func (p Point[T]) Label() string {
	return fmt.Sprintf("%010.5f_%010.5f", p.Lat, p.Lon)
}

func (p Point[T]) Distance(x Point[T]) T {
	return DistanceGeoType[T](p.Lat, p.Lon, x.Lat, x.Lon)
}

func (p Point[T]) Approximately(x Point[T]) T {
	return ApproximateDistanceGeo(p.Lat, p.Lon, x.Lat, x.Lon)
}

// AreaInRange64 is like AreaInRange but using float64
func AreaInRange64[T Float](pt Pair[T], distance T) Rect[T] {
	lat := pt[0]
	lon := pt[1]
	deltaLat := (distance / DegreeToKilometer)
	deltaLon := (LongitudeKilometerDegrees(T(lat), distance))
	min := Pair[T]{lat - deltaLat, lon - deltaLon}
	max := Pair[T]{lat + deltaLat, lon + deltaLon}
	return Rect[T]{min, max}
}

// Closest searches for a matching point within the distance (in Km)
// of the specified point.
// It returns the index of the closest point and the distance from the target
// If nothing is found, it returns the Len() of the points list and -1 distance
//
// NOTE: this is an adaptation of Bestest, but distances are approximated to
//
//	minimize computational load
//
// TODO: the len return is in line w/ Go sort.Search, but perhaps -1 would be better?
// TODO part too: use distance func to share same routine w/ approx and haversine calcs?
func Closest[T Float](g GeoPoints[T], pt Point[T], deltaKm T) (int, T) {
	// Do a binary search to find the "closest" match

	// The point found is not guaranteed to actually be
	// the shortest distance, as it finds the first point
	// that is equal to or *greater* than what is searched for

	// It's possible for the first hit to be significantly further away,
	// whereas an entry before it, while less than the point, is still
	// closer.

	// The final confounding factor is that the data is order by
	// latitude *then* longitude, so the following point could be
	// very close in latitude, the longitude could be much greater.
	// A subsequent point could be 0.000001 degrees latitude
	// further (0.11 m), but have the longitude diff be much less

	x := sort.Search(g.Len(), func(i int) bool {
		h := g.IndexPoint(i)
		return pt.Less(h)
	})

	// did search fail?
	if x == g.Len() {
		return x, -1 //math.MaxFloat64//closest
	}

	// so we either came in exactly on target (not likely),
	// or somewhat past it.
	// Start by going backwards as we already know we probably overshot
	// if we are greater than this then there's no possibility
	// we will be in range continuing on

	// To minimize work done (calculating distance),
	// calculate the furthest away directly by latidude only,
	// as that is (effectively) invariant
	minLat := pt.Lat - (deltaKm / DegreeToKilometer)

	best := x //g.Len()

	//closest := deltaKm + 0.0001 // ensure we have something to best
	counter := 0

	// our first hit is guaranteed to be equal to or *greater* than our
	// requested point.
	//
	// we have to check both above and below the point in question to see
	// which has the closed hit
	this := g.IndexPoint(x)
	dist := this.Approximately(pt)
	closest := dist
	debugf("first hit for %v: %v -- %6d/%6d (%f)", pt, this, x, g.Len(), dist)

	lonKmPerDegree := LookupLonKmPerLat(float64(pt.Lat)) //LookupLonKmPerLatInt(int(pt.Lat))
	deltaLon := (closest / T(lonKmPerDegree))
	lonOutside := func(lon T) bool {
		maxLon := pt.Lon + deltaLon
		minLon := pt.Lon - deltaLon
		return lon < minLon || lon > maxLon
	}

	// work backwards first, as we likely overshot our target
	for i := x - 1; i > 0; i-- {
		counter++
		this = g.IndexPoint(i)
		if this.Lat < minLat {
			//debugf("%v exceeded minimum possible lat: %v", this, minLat)
			break
		}
		if lonOutside(this.Lon) {
			//debugf("below lon outside: %v", this)
			continue
		}
		if dist := pt.Approximately(this); dist < closest {
			closest = dist
			best = i
			minLat = pt.Lat - (closest / DegreeToKilometer)
			deltaLon = (closest / T(lonKmPerDegree))
			//debugf("(%d) MINLAT: %f", counter, minLat)
		}
	}
	/*
	   so we're within range... now we keep looking for anything closer past
	   the initial hit, which will not last long, as we're moving *away* from our destination,
	   and the only likely improvement is if the longitude was off and we
	   find a hit that's closer

	   The closest *possible* will be the same lon but directly above our point,
	   so we calculate what that lat is for the current minimal distance
	   and once we hit pass that lat we know nothing can be closer and we're
	   done with that sweep.
	*/
	maxLat := pt.Lat + (closest / DegreeToKilometer)
	for i := x + 1; i < g.Len(); i++ {
		counter++
		this = g.IndexPoint(i)
		if this.Lat > maxLat {
			//debugf("%v exceeds max lat of %v", this, maxLat)
			break
		}
		if lonOutside(this.Lon) {
			//debugf("above lon outside: %v", this)
			continue
		}
		if dist := this.Approximately(pt); dist < closest {
			best = i
			closest = dist
			maxLat = pt.Lat + (dist / DegreeToKilometer)
		}
	}
	//debugf("Examined %d records", counter)

	return best, closest
}

func between[T Float](check, min, max T) bool {
	return min <= check && check <= max
}

func Within[T Float](lat, lon, minLat, minLon, maxLat, maxLon T) bool {
	return between(lat, minLat, maxLat) && between(lon, minLon, maxLon)
}

func geos[T Float](ss ...string) ([]T, error) {
	ff := make([]T, 0, len(ss))
	for i, s := range ss {
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return nil, fmt.Errorf("(%d/%d): %q ain't a number: %w", i, len(ss), s, err)
		}
		ff = append(ff, T(f))
	}
	return ff, nil
}

const badPrefix = `bad coordinates --`

func coordCheck[T Float](coord ...T) error {
	if len(coord) < 4 {
		return fmt.Errorf("requires 4 coordinates")
	}
	switch {
	// affirm positive lats, negative lons
	case coord[0] == 0 && coord[1] == 0 && coord[2] == 0 && coord[3] == 0:
		return fmt.Errorf(badPrefix + " zero coordinates are not allowed")
	case coord[0] < 0:
		return fmt.Errorf(badPrefix+" latitude %.5f cannot be negative", coord[0])
	case coord[1] > 0:
		return fmt.Errorf(badPrefix+" longitude %.5f must be negative", coord[1])
	case coord[2] < 0:
		return fmt.Errorf(badPrefix+" latitude %.5f cannot be negative", coord[2])
	case coord[3] > 0:
		return fmt.Errorf(badPrefix+" longitude %.5f cannot be negative", coord[3])

	case coord[0] > coord[2]:
		return fmt.Errorf(badPrefix+" latitude %.5f must be less than %.5f", coord[0], coord[2])
		/*
			case coord[0] < AllowedMinLat:
				return fmt.Errorf(badPrefix+" latitude %.5f must be at least %.5f", coord[0], AllowedMinLat)
			case coord[0] > AllowedMaxLat:
				return fmt.Errorf(badPrefix+" latitude %.5f must be less than %.5f", coord[0], AllowedMaxLat)
		*/
	case coord[1] > coord[3]:
		return fmt.Errorf(badPrefix+" longitude %.5f must be less than %.5f", coord[1], coord[3])
		/*
			case coord[1] < AllowedMinLon:
				return fmt.Errorf(badPrefix+" longitude %.5f must be at least %.5f", coord[1], AllowedMinLon)
			case coord[1] > AllowedMaxLon:
				return fmt.Errorf(badPrefix+" longitude %.5f must be less than %.5f", coord[1], AllowedMaxLon)

			case coord[2] > AllowedMaxLat:
				return fmt.Errorf(badPrefix+" latitude %.5f must be less than %.5f", coord[2], AllowedMaxLat)

			case coord[3] > AllowedMaxLon:
				return fmt.Errorf(badPrefix+" longitude %.5f must be less than %.5f", coord[3], AllowedMaxLon)
		*/
	}
	return nil
}

func QueryCoords[T Float](s string) (Pair[T], error) {
	parts := strings.Split(s, ",")
	if len(parts) < 2 {
		parts = strings.Split(s, "/")
		if len(parts) < 2 {
			return Pair[T]{}, ErrInvalidCoordinates
		}
	}
	lat, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return Pair[T]{}, fmt.Errorf("invalid latitude %q -- %w", parts[0], ErrInvalidCoordinates)
	}
	lon, err := strconv.ParseFloat(parts[1], 64)
	if err != nil {
		return Pair[T]{}, fmt.Errorf("invalid longitude %q -- %w", parts[0], ErrInvalidCoordinates)
	}
	return Pair[T]{T(lat), T(lon)}, nil
}

func QueryPoint[T Float](s string) (Point[T], error) {
	pt, err := QueryCoords[T](s)
	return Point[T]{T(pt[0]), T(pt[1])}, err
}

// Bestest searches for a matching point within the distance (in Km)
// of the specified point.
// It returns the index of the closest point and the distance from the target
// If nothing is found, it returns the Len() of the points list and -1 distance
//
// NOTE: this is an adaptation of Bestest, but distances are approximated to
//
//	minimize computational load
//
// TODO: the len return is in line w/ Go sort.Search, but perhaps -1 would be better?
// TODO part too: use distance func to share same routine w/ approx and haversine calcs?
func Bestest[T Float](g GeoPoints[T], pt Point[T], deltaKm T) (int, T) {
	// Do a binary search to find the "closest" match

	// The point found is not guaranteed to actually be
	// the shortest distance, as it finds the first point
	// that is equal to or *greater* than what is searched for

	// It's possible for the first hit to be significantly further away,
	// whereas an entry before it, while less than the point, is still
	// closer.

	// The final confounding factor is that the data is order by
	// latitude *then* longitude, so the following point could be
	// very close in latitude, the longitude could be much greater.
	// A subsequent point could be 0.000001 degrees latitude
	// further (0.11 m), but have the longitude diff be much less

	x := sort.Search(g.Len(), func(i int) bool {
		h := g.IndexPoint(i)
		return pt.Less(h)
	})

	// did search fail?
	if x == g.Len() {
		return x, -1 //math.MaxFloat64//closest
	}

	// so we either came in exactly on target (not likely),
	// or somewhat past it.
	// Start by going backwards as we already know we probably overshot
	// if we are greater than this then there's no possibility
	// we will be in range continuing on

	// To minimize work done (calculating distance),
	// calculate the furthest away directly by latidude only,
	// as that is (effectively) invariant
	minLat := pt.Lat - T(deltaKm/DegreeToKilometer)

	best := g.Len()

	closest := deltaKm + 0.0001 // ensure we have something to best
	counter := 0

	// our first hit is guaranteed to be equal to or *greater* than our
	// requested point.
	//
	// we have to check both above and below the point in question to see
	// which has the closed hit
	this := g.IndexPoint(x)
	dist := this.Distance(pt)
	debugf("first hit: %6d/%6d (%f)", x, g.Len(), dist)
	// if c := T(closest); c < dist {
	if dist < closest {
		closest = dist
		best = x
	}
	debugf("(%d) PT.LAT:%f MINLAT:%f", counter, this.Lat, minLat)

	// only check if lon is in range as well
	/*
		e.g, 100km per degree lon
		10 km range
		10 km / 100km/degree = 0.1 degree delta lon
	*/
	lonKmPerDegree := LonKilos(float64(pt.Lat)) //LookupLonKmPerLatInt(int(pt.Lat))
	deltaLon := T(closest / T(lonKmPerDegree))
	lonOutside := func(lon T) bool {
		maxLon := pt.Lon + deltaLon
		minLon := pt.Lon - deltaLon
		return lon < minLon || lon > maxLon
	}

	// work backwards first, as we likely overshot our target
	for i := x - 1; i > 0; i-- {
		counter++
		this = g.IndexPoint(i)
		if this.Lat < minLat {
			debugf("%v exceeded minimum possible lat: %v", this, minLat)
			break
		}
		if lonOutside(this.Lon) {
			continue
		}
		if dist := pt.Distance(this); dist < closest {
			closest = dist
			best = i
			minLat = pt.Lat - T(closest/DegreeToKilometer)
			deltaLon = T(closest / T(lonKmPerDegree))
			debugf("(%d) MINLAT: %f", counter, minLat)
		}
	}
	/*
	   so we're within range... now we keep looking for anything closer past
	   the initial hit, which will not last long, as we're moving *away* from our destination,
	   and the only likely improvement is if the longitude was off and we
	   find a hit that's closer

	   The closest *possible* will be the same lon but directly above our point,
	   so we calculate what that lat is for the current minimal distance
	   and once we hit pass that lat we know nothing can be closer and we're
	   done with that sweep.
	*/
	maxLat := pt.Lat + T(closest/DegreeToKilometer)
	for i := x + 1; i < g.Len(); i++ {
		counter++
		this = g.IndexPoint(i)
		if this.Lat > maxLat {
			debugf("%v exceeds max lat of %v", this, maxLat)
			break
		}
		if lonOutside(this.Lon) {
			continue
		}
		if dist := this.Distance(pt); dist < closest {
			best = i
			closest = dist
			maxLat = pt.Lat + T(dist/DegreeToKilometer)
		}
	}
	debugf("Examined %d records", counter)

	return best, closest
}

/*
	func ToGeoType(value interface{}) (GeoType, error) {
		switch value := value.(type) {
		case float32:
			return GeoType(value), nil
		case float64:
			return GeoType(value), nil
		case int:
			return GeoType(value), nil
		case int32:
			return GeoType(value), nil
		case int64:
			return GeoType(value), nil
		case string:
			f, err := strconv.ParseFloat(value, 32)
			return GeoType(f), err
		}
		return 0, fmt.Errorf("%v is un unsupported type: %T", value, value)
	}

	func DecodePoint(buf []byte) Point {
		Lat := GeoType(math.Float32frombits(binary.LittleEndian.Uint32(buf)))
		Lon := GeoType(math.Float32frombits(binary.LittleEndian.Uint32(buf[4:])))
		return Point{Lat, Lon}
	}
*/
func DecodePair(buf []byte) Pair[float64] {
	Lat := math.Float64frombits(binary.LittleEndian.Uint64(buf))
	Lon := math.Float64frombits(binary.LittleEndian.Uint64(buf[8:]))
	return Pair[float64]{Lat, Lon}
}
