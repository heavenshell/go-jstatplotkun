package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/codegangsta/cli"
)

// Time format for parse string.
const timeformat = "2006-01-02 15:04:05 -0700"

type appContex struct {
	gc   string
	path string
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

func parseGc(line string) gc {
	lines := pattern.Split(line, -1)
	tof64 := func(value string) float64 {
		var ret, _ = strconv.ParseFloat(value, 64)
		return ret
	}

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

	return gc
}

func parseGcutil(line string) gcutil {
	gcutil := gcutil{}
	return gcutil
}

func read(targetFileName string) []gc {
	_, err := os.Stat(targetFileName)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}

	var fp *os.File
	fp, err = os.Open(targetFileName)
	if err != nil {
		log.Fatal(err)
		os.Exit(1)
	}
	defer fp.Close()

	var lines []string
	scanner := bufio.NewScanner(fp)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	length := len(lines)
	gcs := make([]gc, length - 1)
	for i := 1; i < length; i++ {
		gcs[i - 1] = parseGc(string(lines[i]))
	}

	return gcs
}

func plot(values interface{}) {
	switch data := values.(type) {
	case []gc:
		for _, v := range data {
			fmt.Println(v.YGC)
		}
	case []gcutil:
	default:
		log.Fatalf("Unkown type %v", data)
	}
}

func run(c *cli.Context) {
	jstatOption := c.String("gc")
	jstatPath := c.String("path")

	ctx := appContex{gc: jstatOption, path: jstatPath}
	fmt.Println(ctx)
	gcs := read(jstatPath)
	plot(gcs)

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
			},
		},
	}
	app.Run(os.Args)
}
