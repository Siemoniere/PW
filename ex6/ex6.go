package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	NrOfTravelers = 2
	MinSteps      = 50
	MaxSteps      = 100

	MinDelay = 10 * time.Millisecond
	MaxDelay = 50 * time.Millisecond

	BoardWidth  = NrOfTravelers
	BoardHeight = 4
)

var startTime = time.Now()
var wg sync.WaitGroup

var (
	flag     [2]bool
	turn     int
	flagLock sync.Mutex // guards access to flag and turn
)

type Position struct {
	X int
	Y int
}

type TraceType struct {
	Time_Stamp time.Time
	Id         int
	Position   Position
	Symbol     rune
}

type TraceArray [MaxSteps + 1]TraceType

type TracesSequence struct {
	Last       int
	TraceArray TraceArray
}

func PrintTrace(t TraceType) {
	elapsed := t.Time_Stamp.Sub(startTime).Seconds()
	fmt.Printf("%.6f %d %d %d %c\n", elapsed, t.Id, t.Position.X, t.Position.Y, t.Symbol)
}

func PrintTraces(t TracesSequence) {
	for i := 0; i <= t.Last; i++ {
		PrintTrace(t.TraceArray[i])
	}
}

var reportChannel = make(chan TracesSequence, NrOfTravelers+50)

func printer(done chan struct{}) {
	defer close(done)
	for traces := range reportChannel {
		PrintTraces(traces)
	}
}

func enterPeterson(id int) {
	other := 1 - id
	flagLock.Lock()
	flag[id] = true
	turn = other
	flagLock.Unlock()

	for {
		flagLock.Lock()
		if !flag[other] || turn != other {
			flagLock.Unlock()
			break
		}
		flagLock.Unlock()
		time.Sleep(1 * time.Millisecond) // yield
	}
}

func exitPeterson(id int) {
	flagLock.Lock()
	flag[id] = false
	flagLock.Unlock()
}

func processTask(id int, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
	var traces TracesSequence
	traces.Last = -1

	storeTrace := func(y int, sym rune) {
		if traces.Last < MaxSteps {
			traces.Last++
			traces.TraceArray[traces.Last] = TraceType{
				Time_Stamp: time.Now(),
				Id:         id,
				Position:   Position{X: id, Y: y},
				Symbol:     sym,
			}
		}
	}

	steps := MinSteps + r.Intn(MaxSteps-MinSteps+1)

	for step := 0; step < steps; step++ {
		storeTrace(0, symbol)
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		storeTrace(1, symbol)
		enterPeterson(id)

		storeTrace(2, symbol)
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		storeTrace(3, symbol)
		exitPeterson(id)
		time.Sleep(1 * time.Millisecond)
	}

	reportChannel <- traces
}

func main() {
	symbols := []rune{'A', 'B'}

	printerDone := make(chan struct{})
	go printer(printerDone)

	wg.Add(NrOfTravelers)
	for i := 0; i < NrOfTravelers; i++ {
		go processTask(i, symbols[i])
	}

	wg.Wait()
	close(reportChannel)
	<-printerDone

	fmt.Printf("-1 %d %d %d LOCAL_SECTION;ENTRY_PROTOCOL;CRITICAL_SECTION;EXIT_PROTOCOL;\n",
		NrOfTravelers, BoardWidth, BoardHeight)
}
