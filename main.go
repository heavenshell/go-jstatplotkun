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
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/vdobler/chart"
	"github.com/vdobler/chart/imgg"
)

// Time format for parse string.
const timeformat = "2006-01-02 15:04:05"

// Time zone.
var jst = time.FixedZone("Asia/Tokyo", 9*60*60)

// Application context.
type appContex struct {
	jstatOption     string
	path            string
	startDateTime   *time.Time
	interval        time.Duration
	distPath        string
	logger          *logrus.Logger
	ignoreTimestamp bool
}

type metrix struct {
	points []point
	bars   []bar
}

type point struct {
	title string
	point []chart.EPoint
}

type bar struct {
	title string
	x     []float64
	y     []float64
}

// jstat -gc option.
type gc struct {
	time time.Time
	S0C  float64 `graph:"ScatterChart"`
	S1C  float64 `graph:"ScatterChart"`
	S0U  float64 `graph:"ScatterChart"`
	S1U  float64 `graph:"ScatterChart"`
	EC   float64 `graph:"ScatterChart"`
	EU   float64 `graph:"ScatterChart"`
	OC   float64 `graph:"ScatterChart"`
	OU   float64 `graph:"ScatterChart"`
	PC   float64 `graph:"ScatterChart"`
	PU   float64 `graph:"ScatterChart"`
	YGC  float64 `graph:"ScatterChart"`
	YGCT float64 `graph:"ScatterChart"`
	FGC  float64 `graph:"ScatterChart"`
	FGCT float64 `graph:"ScatterChart"`
	GCT  float64 `graph:"ScatterChart"`
}

// jstat -gcutil option.
type gcutil struct {
	time time.Time
	S0C  float64 `graph:"ScatterChart"`
	S1C  float64 `graph:"ScatterChart"`
	E    float64 `graph:"ScatterChart"`
	O    float64 `graph:"ScatterChart"`
	P    float64 `graph:"ScatterChart"`
	YGC  float64 `graph:"ScatterChart"`
	YGCT float64 `graph:"ScatterChart"`
	FGC  float64 `graph:"ScatterChart"`
	FGCT float64 `graph:"ScatterChart"`
	GCT  float64 `graph:"ScatterChart"`
}

// Regex pattern for parse jstat log file.
var pattern = regexp.MustCompile("\\s+")

// Font
var font *truetype.Font

// Shortcut to convert.
func tof64(v string) float64 {
	var ret, _ = strconv.ParseFloat(v, 64)
	return ret
}

//
func parseGc(line string, start time.Time, ctx appContex) gc {
	lines := pattern.Split(line, -1)
	gc := gc{
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
	if ctx.ignoreTimestamp == false {
		gc.time = start
	}

	return gc
}

// Parse `jstat -gcutil` log file.
// S0     S1     E      O      P     YGC     YGCT    FGC    FGCT     GCT
func parseGcUtil(line string, start time.Time, ctx appContex) gcutil {
	lines := pattern.Split(line, -1)
	gcutil := gcutil{
		S0C:  tof64(lines[0]),
		S1C:  tof64(lines[1]),
		E:    tof64(lines[2]),
		O:    tof64(lines[3]),
		P:    tof64(lines[4]),
		YGC:  tof64(lines[5]),
		YGCT: tof64(lines[6]),
		FGC:  tof64(lines[7]),
		FGCT: tof64(lines[8]),
		GCT:  tof64(lines[9]),
	}
	if ctx.ignoreTimestamp == false {
		gcutil.time = start
	}
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

// Parse jstat -gc or jstat -gcutil.
func parse(lines []string, ctx appContex) (interface{}, error) {
	length := len(lines)

	switch ctx.jstatOption {
	case "gc":
		results := make([]gc, length-1)
		// TODO Refactor
		// 1st line is a title such as `S0C    S1C    S0U    S1U` etc.
		// So start index no is 1.
		for i := 1; i < length; i++ {
			t := ctx.startDateTime.Add(ctx.interval)
			results[i-1] = parseGc(string(lines[i]), t, ctx)
			ctx.startDateTime = &t
		}
		return results, nil
	case "gcutil":
		results := make([]gcutil, length-1)
		// TODO Refactor
		// 1st line is a title such as `S0C    S1C    S0U    S1U` etc.
		// So start index no is 1.
		for i := 1; i < length; i++ {
			t := ctx.startDateTime.Add(ctx.interval)
			results[i-1] = parseGcUtil(string(lines[i]), t, ctx)
			ctx.startDateTime = &t
		}
		return results, nil

	default:
	}

	return nil, fmt.Errorf("can not parse jstat file.")
}

func prepare(values interface{}, graphs []string) metrix {
	p := point{}
	metrix := metrix{}
	points := make([]point, 0, 20)
	for _, g := range graphs {
		switch data := values.(type) {
		case []gc:
			ep := make([]chart.EPoint, 0, 20)
			for _, v := range data {
				r := reflect.ValueOf(v)
				f := reflect.Indirect(r).FieldByName(g)
				st := reflect.TypeOf(v)
				field, _ := st.FieldByName(g)
				graphType := field.Tag.Get("graph")

				if graphType == "ScatterChart" {
					ep = append(ep, chart.EPoint{
						X:      float64(v.time.Unix()),
						Y:      float64(f.Float()),
						DeltaX: math.NaN(),
						DeltaY: math.NaN(),
					})
				}
			}

			p.title = g
			p.point = ep
			points = append(points, p)
		case []gcutil:
			ep := make([]chart.EPoint, 0, 20)
			for _, v := range data {
				r := reflect.ValueOf(v)
				f := reflect.Indirect(r).FieldByName(g)
				st := reflect.TypeOf(v)
				field, _ := st.FieldByName(g)
				graphType := field.Tag.Get("graph")

				if graphType == "ScatterChart" {
					ep = append(ep, chart.EPoint{
						X:      float64(v.time.Unix()),
						Y:      float64(f.Float()),
						DeltaX: math.NaN(),
						DeltaY: math.NaN(),
					})
				}
			}

			p.title = g
			p.point = ep
			points = append(points, p)
		default:
			log.Fatalf("Unkown type %v", data)
		}
		metrix.points = points
	}
	return metrix
}

// Generate scatter chart.
// see https://github.com/mattn/gorecast/blob/master/graph.go#L47
func plotScatter(points []point, title string, ctx appContex) error {
	rgba := image.NewRGBA(image.Rect(0, 0, 1024, 768))
	draw.Draw(rgba, rgba.Bounds(), image.White, image.ZP, draw.Src)
	img := imgg.AddTo(rgba, 0, 0, 1024, 768, color.RGBA{0xff, 0xff, 0xff, 0xff}, font, imgg.ConstructFontSizes(13))

	c := chart.ScatterChart{Title: title}
	c.XRange.TicSetting.Grid = 1
	for _, p := range points {
		c.AddData(p.title, p.point, chart.PlotStyleLines, chart.Style{})
	}

	c.XRange.Time = true
	c.XRange.TicSetting.TFormat = func(t time.Time, td chart.TimeDelta) string {
		return t.Format("15:04:05")
	}
	c.YRange.Label = "count"

	c.Plot(img)

	f, err := os.Create(fmt.Sprintf("%s/%s.png", ctx.distPath, title))
	if err != nil {
		fmt.Println(err)
		return err
	}
	defer f.Close()

	return png.Encode(f, rgba)
}

func setupFont() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Println(err)
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
	logger := setupLog(c.String("verbose"))
	jstatOption := c.String("gc")
	jstatPath := c.String("path")
	start := c.String("date")
	if start == "" {
		start = time.Now().Format(timeformat)
	}
	interval := c.Int("interval") * int(time.Millisecond)

	t, err := time.Parse(timeformat, start)
	if err != nil {
		log.Fatalf("fail to parse. %v", err)
	}
	logger.Info(fmt.Sprintf("reading jstat file is %s", jstatPath))
	distPath, err := filepath.Abs(c.String("output"))
	if err != nil {
		log.Fatalf("%v", err)
	}
	if _, err := os.Stat(distPath); os.IsNotExist(err) {
		os.Mkdir(distPath, 0755)
	}

	ignoreTimestamp := c.Bool("ignore-timestamp")

	ctx := appContex{
		jstatOption:     jstatOption,
		path:            jstatPath,
		interval:        time.Duration(interval),
		startDateTime:   &t,
		distPath:        distPath,
		logger:          logger,
		ignoreTimestamp: ignoreTimestamp,
	}

	logger.Debug("start read jstat file")
	lines, err := read(jstatPath)
	if err != nil {
		log.Fatalf("fail to read file %v", err)
	}

	logger.Debug("start parse jstat file")
	values, err := parse(lines, ctx)
	if err != nil {
		log.Fatal(err)
	}

	charts := map[string][]string{}
	if jstatOption == "gc" {
		charts = map[string][]string{
			"Survivor0": []string{"S0C", "S0U"},
			"Survivor1": []string{"S1C", "S1U"},
			"Eden":      []string{"EC", "EU"},
			"Old":       []string{"OC", "OU"},
			"Perm":      []string{"PC", "PU"},
			"GcCount":   []string{"YGC", "FGC"},
			"Heap":      []string{"S0C", "S0U", "S1C", "S1U", "EC", "EU", "OC", "OU", "PC", "PU"},
			"GcTime":    []string{"YGCT", "FGCT", "FGCT"},
		}
	} else if jstatOption == "gcutil" {
		charts = map[string][]string{
			"Survivor0": []string{"S0C", "S0U"},
			"Survivor1": []string{"S1C", "S1U"},
			"Eden":      []string{"E"},
			"Old":       []string{"O"},
			"Perm":      []string{"P"},
			"GcCount":   []string{"YGC", "FGC"},
			"Heap":      []string{"S0C", "S0U", "S1C", "S1U", "E", "O", "P"},
			"GcTime":    []string{"YGCT", "FGCT", "FGCT"},
		}

	}
	// When using gorutine
	//   1.61s user 0.36s system 86% cpu 2.278 total
	// not using gorutine
	//   1.59s user 0.27s system 101% cpu 1.823 total
	for k, v := range charts {
		logger.Debugf("start parse %s", k)
		metrix := prepare(values, v)
		plotScatter(metrix.points, k, ctx)
	}
	logger.Info("finished")
}

func setupLog(logLevel string) *logrus.Logger {
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		log.Fatalf("Log level error %v", err)
	}
	logger := logrus.Logger{
		Formatter: &logrus.TextFormatter{DisableColors: false},
		Level:     level,
		Out:       os.Stdout,
	}

	return &logger
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
					Usage: "interval mill sec of jstat",
					Value: 1000,
				},
				cli.StringFlag{
					Name:  "output",
					Usage: "output file path",
					Value: "./dist",
				},
				cli.StringFlag{
					Name:  "verbose",
					Usage: "Logger verbose",
					Value: logrus.InfoLevel.String(),
				},
				cli.BoolFlag{
					Name:  "ignore-timestamp",
					Usage: "ignore timestamp",
				},
			},
		},
	}
	app.Run(os.Args)
}
