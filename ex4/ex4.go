package main

import (
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"
)

var WaitGroup sync.WaitGroup

const (
	NrOfProcess = 2

	MinSteps = 50
	MaxSteps = 100

	MinDelay = 10 * time.Millisecond
	MaxDelay = 50 * time.Millisecond

	BoardWidth  = NrOfProcess
	BoardHeight = 4
)

type ProcessState int

const (
	LocalSection ProcessState = iota
	EntryProtocol
	CriticalSection
	ExitProtocol
)

var startTime = time.Now()

// zmienne potrzebne do algorytmu Dekkera
var flag [NrOfProcess]int32
var turn int32

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

type Traces_Sequence_Type struct {
	Last       int
	TraceArray TraceArray
}

func PrintTrace(t TraceType) {
	elapsed := t.Time_Stamp.Sub(startTime).Seconds()
	fmt.Printf("%.6f %d %d %d %c\n", elapsed, t.Id, t.Position.X, t.Position.Y, t.Symbol)
}

func PrintTraces(t Traces_Sequence_Type) {
	for i := 0; i <= t.Last; i++ {
		PrintTrace(t.TraceArray[i])
	}
}

var reportChannel = make(chan Traces_Sequence_Type, NrOfProcess)

func printer() {
	for i := 0; i < NrOfProcess; i++ {
		traces := <-reportChannel
		PrintTraces(traces)
	}

	fmt.Printf("-1 %d %d %d ", NrOfProcess, BoardWidth, BoardHeight)

	states := []string{"Local_Section", "Entry_Protocol", "Critical_Section", "Exit_Protocol"}
	for _, state := range states {
		fmt.Printf("%s;", state)
	}

	fmt.Println("EXTRA_LABEL;")
	WaitGroup.Done()
}

type Process struct {
	Id       int
	Symbol   rune
	Position Position
}

func process(id int, symbol rune, seed int) {
	defer func() {
		atomic.StoreInt32(&flag[id], 0)
		atomic.StoreInt32(&turn, int32(1-id))
	}()

	defer WaitGroup.Done()
	r := rand.New(rand.NewSource(int64(seed)))

	var state ProcessState = LocalSection

	var process Process
	process.Id = id
	process.Symbol = symbol
	process.Position.X = id
	process.Position.Y = int(state)

	var traces Traces_Sequence_Type
	traces.Last = -1

	storeTrace := func() {
		process.Position.Y = int(state)
		timeStamp := time.Since(startTime)
		traces.Last++
		traces.TraceArray[traces.Last] = TraceType{
			Time_Stamp: startTime.Add(timeStamp),
			Id:         process.Id,
			Position:   process.Position,
			Symbol:     process.Symbol,
		}
	}

	storeTrace()

	nrOfSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)

	for i := 0; i < nrOfSteps/4-1; i++ {
		state = LocalSection
		delay := MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay)))
		time.Sleep(delay)
		state = EntryProtocol
		storeTrace()
		atomic.StoreInt32(&flag[id], 1)
		for atomic.LoadInt32(&flag[1-id]) == 1 {
			if atomic.LoadInt32(&turn) == int32(1-id) {
				atomic.StoreInt32(&flag[id], 0)
				for atomic.LoadInt32(&turn) != int32(id) {
					time.Sleep(1 * time.Millisecond)
				}
				atomic.StoreInt32(&flag[id], 1)
			}
		}
		delay = MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay)))
		time.Sleep(delay)
		state = CriticalSection
		storeTrace()
		delay = MinDelay
		time.Sleep(delay)

		state = ExitProtocol
		storeTrace()
		atomic.StoreInt32(&flag[id], 0)
		atomic.StoreInt32(&turn, int32(1-id))
		delay = MinDelay
		time.Sleep(delay)
		state = LocalSection
		storeTrace()
	}
	reportChannel <- traces
}

func main() {
	WaitGroup.Add(1)
	go printer()
	symbols := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O'}
	for i := 0; i < NrOfProcess; i++ {
		WaitGroup.Add(1)
		go process(i, symbols[i], i)
	}
	WaitGroup.Wait()
}
