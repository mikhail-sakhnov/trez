package trez

type Gravity int

const (
	CENTER Gravity = iota
	NORTH
	SOUTH
	WEST
	EAST
	NORTH_WEST
	NORTH_EAST
	SOUTH_WEST
	SOUTH_EAST
)

type Format int

const (
	JPEG Format = iota
	WEBP
	PNG
)

type Algo int

const (
	FIT Algo = iota
	FILL
)

type Options struct {
	Width       int
	Height      int
	MaxSide     int
	MaxWidth    int
	MaxHeight   int
	Algo        Algo
	Background  [3]int
	Gravity     Gravity
	Format      Format
	Quality     int
	Progressive bool
}

func (a Algo) String() string {
	switch a {
	case FIT:
		return "fit"
	case FILL:
		return "fill"
	default:
		return "invalid"
	}
}
