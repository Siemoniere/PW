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
	NrOfProcess = 15

	MinSteps = 50
	MaxSteps = 100

	MinDelay = 10 * time.Millisecond
	MaxDelay = 50 * time.Millisecond

	BoardWidth  = NrOfProcess
	BoardHeight = 4
)

// zmienne potrzebne do algorytmu Piekarnianego
var Flag [NrOfProcess]int32
var Number [NrOfProcess]int32
var maxUsedTicket int32 = 0

func findMax(arr []int32) int32 {
	maxVal := atomic.LoadInt32(&arr[0])
	for i := 1; i < len(arr); i++ {
		val := atomic.LoadInt32(&arr[i])
		if val > maxVal {
			maxVal = val
		}
	}
	return maxVal
}
func updateMaxTicket(val int32) {
	for {
		current := atomic.LoadInt32(&maxUsedTicket)
		if val <= current {
			return
		}
		if atomic.CompareAndSwapInt32(&maxUsedTicket, current, val) {
			return
		}
	}
}

type ProcessState int

const (
	LocalSection ProcessState = iota
	EntryProtocol
	CriticalSection
	ExitProtocol
)

var startTime = time.Now()

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
	defer WaitGroup.Done()
	for i := 0; i < NrOfProcess; i++ {
		traces := <-reportChannel
		PrintTraces(traces)
	}

	fmt.Printf("-1 %d %d %d ", NrOfProcess, BoardWidth, BoardHeight)

	states := []string{"Local_Section", "Entry_Protocol", "Critical_Section", "Exit_Protocol"}
	for _, state := range states {
		fmt.Printf("%s;", state)
	}

	fmt.Println("MAX_TICKET = ", atomic.LoadInt32(&maxUsedTicket), ";")

}

type Process struct {
	Id       int
	Symbol   rune
	Position Position
}

func process(id int, symbol rune, seed int) {
	defer WaitGroup.Done()

	r := rand.New(rand.NewSource(int64(seed)))

	var state ProcessState = LocalSection

	process := Process{
		Id:     id,
		Symbol: symbol,
		Position: Position{
			X: id,
			Y: int(state),
		},
	}

	var traces Traces_Sequence_Type
	traces.Last = -1
	var lastState ProcessState = -1
	storeTrace := func() {
		if state != lastState && traces.Last < MaxSteps {
			process.Position.Y = int(state)
			timeStamp := time.Since(startTime)
			traces.Last++
			traces.TraceArray[traces.Last] = TraceType{
				Time_Stamp: startTime.Add(timeStamp),
				Id:         process.Id,
				Position:   process.Position,
				Symbol:     process.Symbol,
			}
			lastState = state
		}
	}

	storeTrace() // trace initial position

	nrOfSteps := MinSteps + r.Intn(MaxSteps-MinSteps+1)
	nrOfSteps = nrOfSteps/4 - 1
	for step := 0; step < nrOfSteps; step++ {
		// LOCAL_SECTION
		state = LocalSection
		storeTrace()
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		// ENTRY_PROTOCOL
		state = EntryProtocol
		storeTrace()

		// Entry protocol logic (choosing ticket number)
		atomic.StoreInt32(&Flag[id], 1)
		maxTicket := findMax(Number[:])
		newTicket := maxTicket + 1
		atomic.StoreInt32(&Number[id], newTicket)
		updateMaxTicket(newTicket)
		atomic.StoreInt32(&Flag[id], 0)

		// Wait for other processes
		for j := 0; j < NrOfProcess; j++ {
			if j == id {
				continue
			}

			// Wait while process j is choosing number
			for atomic.LoadInt32(&Flag[j]) == 1 {
				time.Sleep(1 * time.Millisecond)
			}

			// Wait while j has ticket and j's ticket < id's or equal but j < id
			for atomic.LoadInt32(&Number[j]) != 0 &&
				(atomic.LoadInt32(&Number[j]) < atomic.LoadInt32(&Number[id]) ||
					(atomic.LoadInt32(&Number[j]) == atomic.LoadInt32(&Number[id]) && j < id)) {
				time.Sleep(10 * time.Millisecond)
			}
		}

		// CRITICAL_SECTION
		state = CriticalSection
		storeTrace()
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		// EXIT_PROTOCOL
		state = ExitProtocol
		storeTrace()

		// Reset ticket and update max if needed
		currentTicket := atomic.LoadInt32(&Number[id])
		if currentTicket > atomic.LoadInt32(&Number[id]) {
			// optional: update some global max if you keep track (not mandatory here)

		}
		atomic.StoreInt32(&Number[id], 0)
		time.Sleep(1 * time.Millisecond)
		state = LocalSection
		storeTrace()

		time.Sleep(1 * time.Millisecond)
	}
	reportChannel <- traces
}

func main() {
	//fmt.Println("Poczatkoa tablica: ", Flag)
	WaitGroup.Add(1)
	go printer()
	symbols := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O'}
	for i := 0; i < NrOfProcess; i++ {
		WaitGroup.Add(1)
		go process(i, symbols[i], i)
	}
	WaitGroup.Wait()
}
