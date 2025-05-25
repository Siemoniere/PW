package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	NrOfTravelers = 2 // Dekker's works only for 2 processes
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
	flag       [2]bool
	turn       int
	dekkerLock sync.Mutex // guards access to flag and turn
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

func enterDekker(id int, r *rand.Rand) {
	other := 1 - id
	dekkerLock.Lock()
	flag[id] = true
	dekkerLock.Unlock()

	for {
		dekkerLock.Lock()
		if flag[other] {
			if turn == other {
				flag[id] = false
				dekkerLock.Unlock()
				time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))
				dekkerLock.Lock()
				flag[id] = true
				dekkerLock.Unlock()
				continue
			}
		} else {
			dekkerLock.Unlock()
			break
		}
		dekkerLock.Unlock()
	}
}

func exitDekker(id int) {
	other := 1 - id
	dekkerLock.Lock()
	turn = other
	flag[id] = false
	dekkerLock.Unlock()
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
		enterDekker(id, r)

		storeTrace(2, symbol)
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		storeTrace(3, symbol)
		exitDekker(id)
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
