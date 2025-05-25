package main

import (
	"fmt"
	"math/rand"
	"sync"
	"time"
)

const (
	NrOfTravelers = 15
	MinSteps      = 50
	MaxSteps      = 100

	MinDelay = 10 * time.Millisecond
	MaxDelay = 50 * time.Millisecond

	BoardWidth  = NrOfTravelers
	BoardHeight = 4
)

var startTime = time.Now()
var wg sync.WaitGroup

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

type Bakery struct {
	choosing      []bool
	number        []int
	maxUsedTicket int
	lock          sync.Mutex
}

func NewBakery(n int) *Bakery {
	return &Bakery{
		choosing: make([]bool, n),
		number:   make([]int, n),
	}
}

func (b *Bakery) SetChoosing(i int, val bool) {
	b.lock.Lock()
	b.choosing[i] = val
	b.lock.Unlock()
}

func (b *Bakery) IsChoosing(i int) bool {
	b.lock.Lock()
	val := b.choosing[i]
	b.lock.Unlock()
	return val
}

func (b *Bakery) SetNumber(i int, val int) {
	b.lock.Lock()
	b.number[i] = val
	b.lock.Unlock()
}

func (b *Bakery) GetNumber(i int) int {
	b.lock.Lock()
	val := b.number[i]
	b.lock.Unlock()
	return val
}

func (b *Bakery) MaxTicket() int {
	b.lock.Lock()
	defer b.lock.Unlock()
	max := 0
	for _, v := range b.number {
		if v > max {
			max = v
		}
	}
	return max
}

// UpdateMax aktualizuje globalny max ticket jeśli nowa wartość jest większa
func (b *Bakery) UpdateMax(val int) {
	b.lock.Lock()
	if val > b.maxUsedTicket {
		b.maxUsedTicket = val
	}
	b.lock.Unlock()
}

func (b *Bakery) GetMax() int {
	b.lock.Lock()
	val := b.maxUsedTicket
	b.lock.Unlock()
	return val
}

// ResetTicket zeruje ticket procesu i aktualizuje maxUsedTicket jeśli trzeba
func (b *Bakery) ResetTicket(processId int) {
	b.lock.Lock()
	ticket := b.number[processId]
	if ticket > b.maxUsedTicket {
		b.maxUsedTicket = ticket
	}
	b.number[processId] = 0
	b.lock.Unlock()
}

var bakery = NewBakery(NrOfTravelers)

func processTask(id int, symbol rune) {
	defer wg.Done()
	r := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id)))
	var traces TracesSequence
	traces.Last = -1
	//pos := Position{X: id, Y: 0} // starting at Local_Section = 0

	// pomocnicza funkcja do zapisu śladu
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

	// liczba kroków do wykonania
	steps := MinSteps + r.Intn(MaxSteps-MinSteps+1)

	for step := 0; step < steps; step++ {
		// Local Section
		storeTrace(0, symbol)
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		// Entry protocol
		storeTrace(1, symbol)
		bakery.SetChoosing(id, true)
		maxTicket := bakery.MaxTicket()
		newTicket := maxTicket + 1
		bakery.SetNumber(id, newTicket)
		bakery.UpdateMax(newTicket)
		bakery.SetChoosing(id, false)

		for j := 0; j < NrOfTravelers; j++ {
			for bakery.IsChoosing(j) {
				// czekaj
			}
			for bakery.GetNumber(j) != 0 &&
				(bakery.GetNumber(j) < bakery.GetNumber(id) ||
					(bakery.GetNumber(j) == bakery.GetNumber(id) && j < id)) {
				// czekaj
			}
		}

		// Critical Section
		storeTrace(2, symbol)
		time.Sleep(MinDelay + time.Duration(r.Int63n(int64(MaxDelay-MinDelay))))

		// Exit protocol
		storeTrace(3, symbol)
		bakery.ResetTicket(id)
		time.Sleep(1 * time.Millisecond)

	}

	reportChannel <- traces
}

func main() {

	symbols := []rune{'A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O'}

	printerDone := make(chan struct{})
	go printer(printerDone)

	wg.Add(NrOfTravelers)
	for i := 0; i < NrOfTravelers; i++ {
		go processTask(i, symbols[i])
	}
	wg.Wait()
	close(reportChannel)
	<-printerDone
	fmt.Printf("-1 %d %d %d LOCAL_SECTION;ENTRY_PROTOCOL;CRITICAL_SECTION;EXIT_PROTOCOL;MAX_TICKET=%d\n", NrOfTravelers, BoardWidth, BoardHeight, bakery.GetMax())

}
