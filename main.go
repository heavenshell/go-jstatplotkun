package main

import (
	"bufio"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"time"

	"code.google.com/p/freetype-go/freetype"
	"code.google.com/p/freetype-go/freetype/truetype"
	"github.com/codegangsta/cli"
	"github.com/vdobler/chart"
	"github.com/vdobler/chart/imgg"
)

// Time format for parse string.
const timeformat = "2006-01-02 15:04:05"

var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

type appContex struct {
	gc            string
	path          string
	startDateTime *time.Time
	interval      time.Duration
}

// jstat -gc option.
type gc struct {
	time time.Time
	S0C  float64
	S1C  float64
	S0U  float64
	S1U  float64
	EC   float64
	EU   float64
	OC   float64
	OU   float64
	PC   float64
	PU   float64
	YGC  float64
	YGCT float64
	FGC  float64
	FGCT float64
	GCT  float64
}

// jstat -gcutil option.
type gcutil struct {
	S0C  float64
	S1C  float64
	E    float64
	O    float64
	P    float64
	YGC  float64
	YGCT float64
	FGC  float64
	FGCT float64
	GCT  float64
}

var pattern = regexp.MustCompile("\\s+")

// Font
var font *truetype.Font

func tof64(v string) float64 {
	var ret, _ = strconv.ParseFloat(v, 64)
	return ret
}

func parseGc(line string, start time.Time) gc {
	lines := pattern.Split(line, -1)
	gc := gc{
		time: start,
		S0C:  tof64(lines[0]),
		S1C:  tof64(lines[1]),
		S0U:  tof64(lines[2]),
		S1U:  tof64(lines[3]),
		EC:   tof64(lines[4]),
		EU:   tof64(lines[5]),
		OC:   tof64(lines[6]),
		OU:   tof64(lines[7]),
		PC:   tof64(lines[8]),
		PU:   tof64(lines[9]),
		YGC:  tof64(lines[10]),
		YGCT: tof64(lines[11]),
		FGC:  tof64(lines[12]),
		FGCT: tof64(lines[13]),
		GCT:  tof64(lines[14]),
	}

	return gc
}

func parseGcutil(line string) gcutil {
	gcutil := gcutil{}
	return gcutil
}

func read(file string) ([]string, error) {
	_, err := os.Stat(file)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	var fp *os.File
	fp, err = os.Open(file)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	defer fp.Close()

	var lines []string
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

func parse(lines []string, ctx appContex) []gc {
	length := len(lines)
	gcs := make([]gc, length-1)
	// 1st line is a title such as `S0C    S1C    S0U    S1U` etc.
	// So start index 1.
	for i := 1; i < length; i++ {
		t := ctx.startDateTime.Add(ctx.interval)
		gcs[i-1] = parseGc(string(lines[i]), t)
		ctx.startDateTime = &t
	}

	return gcs
}

func prepare(values interface{}, graphs []string) []map[string][]chart.EPoint {
	results := make([]map[string][]chart.EPoint, 0, 20)
	for _, g := range graphs {
		ep := make([]chart.EPoint, 0, 20)
		switch data := values.(type) {
		case []gc:
			for _, v := range data {
				r := reflect.ValueOf(v)
				f := reflect.Indirect(r).FieldByName(g)
				ep = append(ep, chart.EPoint{
					X:      float64(v.time.Unix()),
					Y:      float64(f.Float()),
					DeltaX: math.NaN(),
					DeltaY: math.NaN(),
				})
			}
			var m = map[string][]chart.EPoint{g: ep}
			results = append(results, m)
		default:
			log.Fatalf("Unkown type %v", data)
		}
	}
	return results
}

// eps is {"S0U": [chart.EPoint{X: xx, Y: yy, DeltaX: aa, DeltaY: bb}]}
// see https://github.com/mattn/gorecast/blob/master/graph.go#L47
func plot(eps []map[string][]chart.EPoint, title string) error {
	rgba := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	draw.Draw(rgba, rgba.Bounds(), image.White, image.ZP, draw.Src)
	img := imgg.AddTo(rgba, 0, 0, 1024, 768, color.RGBA{0xff, 0xff, 0xff, 0xff}, font, imgg.ConstructFontSizes(13))

	c := chart.ScatterChart{Title: title}
	c.XRange.TicSetting.Grid = 1
	for _, data := range eps {
		for k, v := range data {
			c.AddData(k, v, chart.PlotStyleLines, chart.Style{})
		}
	}

	c.XRange.Time = true
	c.XRange.TicSetting.TFormat = func(t time.Time, td chart.TimeDelta) string {
		return t.Format("15:04:05")
	}
	c.YRange.Label = "count"

	c.Plot(img)

	f, err := os.Create(filepath.Join("./", fmt.Sprintf("%s.png", title)))
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, rgba)
}

func setupFont() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
		return
	}
	b, err := ioutil.ReadFile(filepath.Join(cwd, "fonts", "ipaexg.ttf"))
	if err != nil {
		log.Fatal(err)
	}
	font, err = freetype.ParseFont(b)
	if err != nil {
		log.Println(err)
	}
}

func run(c *cli.Context) {
	jstatOption := c.String("gc")
	jstatPath := c.String("path")
	start := c.String("date")
	if start == "" {
		start = time.Now().Format(timeformat)
	}
	interval := c.Int("interval") * int(time.Millisecond)

	ctx := appContex{gc: jstatOption, path: jstatPath}

	t, err := time.Parse(timeformat, start)
	if err != nil {
		log.Fatalf("fail to parse. %v", err)
	}
	ctx.interval = time.Duration(interval)
	ctx.startDateTime = &t

	lines, err := read(jstatPath)
	if err != nil {
		log.Fatalf("fail to read file %v", err)
	}

	gcs := parse(lines, ctx)

	categories := map[string][]string{
		"Survivor0": []string{"S0C", "S0U"},
		"Survivor1": []string{"S1C", "S1U"},
		"Eden":      []string{"EC", "EU"},
		"Old":       []string{"OC", "OU"},
		"Perm":      []string{"PC", "PU"},
		"GcCount":   []string{"YGC", "FGC"},
		"Heap":      []string{"S0C", "S0U", "S1C", "S1U", "EC", "EU", "OC", "OU", "PC", "PU"},
	}

	for k, v := range categories {
		eps := prepare(gcs, v)
		plot(eps, k)
	}
}

func main() {
	app := cli.NewApp()
	app.Name = "jstatplotkun"
	app.Usage = ""
	app.EnableBashCompletion = true
	app.Commands = []cli.Command{
		{
			Name:   "jstat",
			Usage:  "Read gc.log",
			Action: run,
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "gc",
					Usage: "jstat option",
					Value: "gc",
				},
				cli.StringFlag{
					Name:  "path",
					Usage: "Path to jstat file",
				},
				cli.StringFlag{
					Name:  "date",
					Usage: "start time",
				},
				cli.IntFlag{
					Name:  "interval",
					Usage: "interval of jstat",
					Value: 1000,
				},
			},
		},
	}
	app.Run(os.Args)
}
