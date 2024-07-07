package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/nsf/termbox-go"
)

const (
	usage = `
 countdown [-up] [-t] [-n] <duration>

 Usage
  countdown 25s
  countdown 14:15
  countdown 02:15PM
  countdown -t Tag -n "Notes for the activity" 10m

 Flags
`
	tick         = time.Second
	inputDelayMS = 500 * time.Millisecond
)

var (
	timer          *time.Timer
	ticker         *time.Ticker
	queues         chan termbox.Event
	w, h           int
	inputStartTime time.Time
	isPaused       bool
	tag            string
	notes          string
	logPath       string
)

func main() {

	countUp := flag.Bool("up", false, "count up from zero")
	tag := flag.String("t", "Unset", "The tag for this activity")
	notes := flag.String("n", "", "Notes for this activity")
	logPath := flag.String("f", os.Getenv("COUNTDOWN_LOG_PATH"), "The log path")
	flag.Parse()

	if *logPath == "" {
		fmt.Println("No file argument given, set COUNTDOWN_LOG_PATH env variable or provide a file as -f argument.")
		os.Exit(2)
	}
	_, err := os.OpenFile(*logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		fmt.Println("There was a problem accessing " + *logPath)
		os.Exit(2)
	}

	args := flag.Args()
	if len(args) != 1 {
		stderr(usage)
		flag.PrintDefaults()
		os.Exit(2)
	}
	timeLeft, err := parseTime(args[0])

	if err != nil {
		timeLeft, err = time.ParseDuration(args[0])
		if err != nil {
			stderr("error: invalid duration or time: %v\n", args[0])
			os.Exit(2)
		}
	}

	err = termbox.Init()
	if err != nil {
		panic(err)
	}

	queues = make(chan termbox.Event)
	go func() {
		for {
			queues <- termbox.PollEvent()
		}
	}()
	countdown(timeLeft, *countUp, *tag, *notes, *logPath)
}

func start(d time.Duration) {
	timer = time.NewTimer(d)
	ticker = time.NewTicker(tick)
}

func stop() {
	timer.Stop()
	ticker.Stop()
}

func durationToDraw(timeLeft, totalDuration time.Duration, countUp bool) time.Duration {
	if countUp {
		return totalDuration - timeLeft
	}
	return timeLeft
}

func countdown(totalDuration time.Duration, countUp bool, tag string, notes string, logPath string) {
	timeLeft := totalDuration
	var exitCode int
	isPaused = false
	w, h = termbox.Size()
	start(timeLeft)
	appendToLog("i", tag, notes, logPath)

	draw(durationToDraw(timeLeft, totalDuration, countUp), w, h)

loop:
	for {
		select {
		case ev := <-queues:
			if ev.Key == termbox.KeyEsc || ev.Key == termbox.KeyCtrlC {
				exitCode = 1
				appendToLog("o", tag, "", logPath)
				break loop
			}

			if pressTime := time.Now(); ev.Key == termbox.KeySpace && pressTime.Sub(inputStartTime) > inputDelayMS {
				if isPaused {
					start(timeLeft)
					appendToLog("u", tag, "", logPath)
					draw(durationToDraw(timeLeft, totalDuration, countUp), w, h)
				} else {
					stop()
					appendToLog("p", tag, "", logPath)
					drawPause(w, h)
				}

				isPaused = !isPaused
				inputStartTime = time.Now()
			}

			if ev.Type == termbox.EventResize {
				w, h = termbox.Size()
				draw(durationToDraw(timeLeft, totalDuration, countUp), w, h)

				if isPaused {
					drawPause(w, h)
				}
			}
		case <-ticker.C:
			timeLeft -= tick
			draw(durationToDraw(timeLeft, totalDuration, countUp), w, h)
		case <-timer.C:
			appendToLog("o", tag, "", logPath)
			break loop
		}
	}

	termbox.Close()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func draw(d time.Duration, w int, h int) {
	clear()

	str := format(d)
	text := toText(str)

	startX, startY := w/2-text.width()/2, h/2-text.height()/2

	x, y := startX, startY
	for _, s := range text {
		echo(s, x, y)
		x += s.width()
	}

	flush()
}

func drawPause(w int, h int) {
	startX := w/2 - pausedText.width()/2
	startY := h * 3 / 4

	echo(pausedText, startX, startY)
	flush()
}

func format(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h < 1 {
		return fmt.Sprintf("%02d:%02d", m, s)
	}
	return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
}

func appendToLog(state string, tag string, notes string, logPath string){
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		stderr("There was a problem accessing " + logPath)
		os.Exit(2)
	}
	defer f.Close()

	var log string = state + " " + time.Now().Format("2006-01-02 15:04:05") + " " +tag + "  " + notes + "\n"

	if _, err = f.WriteString(log); err != nil {
		stderr("There was a problem writing to " + logPath)
		os.Exit(2)
	}
}

func parseTime(date string) (time.Duration, error) {
	targetTime, err := time.Parse(time.Kitchen, strings.ToUpper(date))
	if err != nil {
		targetTime, err = time.Parse("15:04", date)
		if err != nil {
			return time.Duration(0), err
		}
	}

	now := time.Now()
	originTime := time.Date(0, time.January, 1, now.Hour(), now.Minute(), now.Second(), 0, time.UTC)

	// The time of day has already passed, so target tomorrow.
	if targetTime.Before(originTime) {
		targetTime = targetTime.AddDate(0, 0, 1)
	}

	duration := targetTime.Sub(originTime)

	return duration, err
}
