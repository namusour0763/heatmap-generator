package main

import (
	"encoding/csv"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	cellSize     = 20
	cellGap      = 2
	numWeeks     = 53
	daysInWeek   = 7
	monthsInYear = 12
	legendWidth  = 200
	titleHeight  = 40
	monthHeight  = 20
)

var baseColors = []color.RGBA{
	{R: 235, G: 237, B: 240, A: 255}, // 0 tweets (always light gray)
	{R: 155, G: 233, B: 168, A: 255},
	{R: 64, G: 196, B: 99, A: 255},
	{R: 48, G: 161, B: 78, A: 255},
	{R: 33, G: 110, B: 57, A: 255},
}

type DailyTweet struct {
	Date  time.Time
	Count int
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("Usage: go run main.go input.csv output.png")
	}

	inputFile := os.Args[1]
	outputFile := os.Args[2]

	tweets, err := readCSV(inputFile)
	if err != nil {
		log.Fatal(err)
	}

	img, err := generateHeatmap(tweets)
	if err != nil {
		log.Fatal(err)
	}

	if err := savePNG(img, outputFile); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Heatmap generated successfully:", outputFile)
}

func readCSV(filename string) ([]DailyTweet, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	reader := csv.NewReader(file)

	// Skip header
	if _, err := reader.Read(); err != nil {
		return nil, err
	}

	var tweets []DailyTweet
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		date, err := time.Parse("20060102", record[0])
		if err != nil {
			return nil, err
		}

		count, err := strconv.Atoi(record[1])
		if err != nil {
			return nil, err
		}

		tweets = append(tweets, DailyTweet{Date: date, Count: count})
	}

	return tweets, nil
}

func generateHeatmap(tweets []DailyTweet) (*image.RGBA, error) {
	width := cellSize*numWeeks + cellGap*(numWeeks-1) + legendWidth
	height := cellSize*daysInWeek + cellGap*(daysInWeek-1) + titleHeight + monthHeight

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(img, img.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)

	tweetMap := make(map[time.Time]int)
	var counts []int
	maxCount := 0
	for _, tweet := range tweets {
		tweetMap[tweet.Date] = tweet.Count
		counts = append(counts, tweet.Count)
		if tweet.Count > maxCount {
			maxCount = tweet.Count
		}
	}

	sort.Ints(counts)
	thresholds := calculateThresholds(counts)

	lastTweetDate := tweets[len(tweets)-1].Date
	startDate := lastTweetDate.AddDate(-1, 0, 1)

	for week := 0; week < numWeeks; week++ {
		for day := 0; day < daysInWeek; day++ {
			date := startDate.AddDate(0, 0, week*7+day)
			count := tweetMap[date]

			colorIndex := getColorIndex(count, thresholds)

			x := week * (cellSize + cellGap)
			y := day*(cellSize+cellGap) + titleHeight + monthHeight

			drawRect(img, x, y, cellSize, cellSize, baseColors[colorIndex])
		}
	}

	drawTitle(img, "Tweet Activity Heatmap")
	drawMonths(img, startDate)
	if err := drawLegend(img, thresholds); err != nil {
		return nil, err
	}

	return img, nil
}

func calculateThresholds(counts []int) []int {
	if len(counts) == 0 {
		return []int{0, 0, 0, 0}
	}

	maxCount := counts[len(counts)-1]
	thresholds := make([]int, len(baseColors)-1)
	for i := range thresholds {
		thresholds[i] = int(math.Ceil(float64(maxCount) * float64(i+1) / float64(len(baseColors))))
	}

	return thresholds
}

func getColorIndex(count int, thresholds []int) int {
	for i, threshold := range thresholds {
		if count <= threshold {
			return i
		}
	}
	return len(baseColors) - 1
}

func drawRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			img.Set(x+dx, y+dy, c)
		}
	}
}

func drawTitle(img *image.RGBA, title string) {
	d := &font.Drawer{
		Dst:  img,
		Src:  image.NewUniform(color.Black),
		Face: basicfont.Face7x13,
		Dot:  fixed.Point26_6{X: fixed.Int26_6(10 << 6), Y: fixed.Int26_6(25 << 6)},
	}
	d.DrawString(title)
}

func drawMonths(img *image.RGBA, startDate time.Time) {
	monthNames := []string{"Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"}
	currentMonth := startDate.Month()
	for week := 0; week < numWeeks; week++ {
		date := startDate.AddDate(0, 0, week*7)
		if date.Month() != currentMonth {
			currentMonth = date.Month()
			x := week * (cellSize + cellGap)
			d := &font.Drawer{
				Dst:  img,
				Src:  image.NewUniform(color.Black),
				Face: basicfont.Face7x13,
				Dot:  fixed.Point26_6{X: fixed.Int26_6(x << 6), Y: fixed.Int26_6((titleHeight + 15) << 6)},
			}
			d.DrawString(monthNames[currentMonth-1])
		}
	}
}

func drawLegend(img *image.RGBA, thresholds []int) error {
	legendX := cellSize*numWeeks + cellGap*(numWeeks-1) + 10
	legendY := titleHeight + monthHeight + 10

	for i := 0; i < len(baseColors); i++ {
		drawRect(img, legendX, legendY+i*30, 20, 20, baseColors[i])
		var label string
		if i == 0 {
			label = "0"
		} else if i == len(baseColors)-1 {
			label = fmt.Sprintf("%d+", thresholds[i-1]+1)
		} else {
			if i-1 >= len(thresholds) {
				return fmt.Errorf("index out of range for thresholds: %d", i-1)
			}
			if i >= len(thresholds) {
				return fmt.Errorf("index out of range for thresholds: %d", i)
			}
			label = fmt.Sprintf("%d-%d", thresholds[i-1]+1, thresholds[i])
		}

		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(color.Black),
			Face: basicfont.Face7x13,
			Dot:  fixed.Point26_6{X: fixed.Int26_6((legendX + 30) << 6), Y: fixed.Int26_6((legendY + i*30 + 15) << 6)},
		}
		d.DrawString(label)
	}

	return nil
}

func savePNG(img *image.RGBA, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	return png.Encode(file, img)
}
