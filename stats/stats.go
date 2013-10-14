package stats

import(
	"github.com/trendrr/goshire/dynmap"
	"github.com/trendrr/goshire/timeamount"
	"time"
	"log"

)

// A simple stats collector

type Stats struct {
	itemChan chan statsItem
	getChan chan getRequest
	Persister StatsPersister
	items map[timeamount.TimeAmount]*StatsSave

}

type StatsPersister interface {
	Persist(t timeamount.TimeAmount, val dynmap.DynMap )

}

type statsItemType int
const (
	SET statsItemType = 0
	INC                = 1
)

type getRequest struct {
	resultChan chan map[timeamount.TimeAmount]*StatsSave
}

type statsItem struct {
	Key string
	Val int64
	typ statsItemType
}

type StatsSave struct {
	Epoch int64
	TimeAmount timeamount.TimeAmount
	Values *dynmap.DynMap
}

// Creates a new Stats tracker. caller must still envoke the Start function
// timeamounts should be in the form "{num} {timeframe}" 
// example 
// NewStats("1 minute", "30 minute", "1 day")
func New(timeamounts ...string) (*Stats, error) {
	s := &Stats{
		itemChan : make(chan statsItem, 500),
		items : make(map[timeamount.TimeAmount]*StatsSave),
	}
	for _,ta := range(timeamounts) {
		t, err := timeamount.Parse(ta)
		if err != nil {
			return nil, err
		}
		s.items[t] = &StatsSave{
				Epoch : t.ToTrendrrEpoch(time.Now()),
				TimeAmount : t,
				Values : dynmap.New(),
		}
	}
	return s, nil
}

// Sets the given key with the given value.
func (this *Stats) Set(key string, val int64) {
	select {
	case this.itemChan <- statsItem{Key : key, Val : int64(val), typ : SET} :
	default :
		log.Printf( "Could not Inc Key: %s Val: %d", key, val)
	}
}

func (this *Stats) Inc(key string, val int) {
	select {
	case this.itemChan <- statsItem{Key : key, Val : int64(val), typ : INC} :
	default :
		log.Printf( "Could not Inc Key: %s Val: %d", key, val)
	}
}

func (this *Stats) Get() map[timeamount.TimeAmount]*StatsSave {
	greq := getRequest{
		resultChan : make(chan map[timeamount.TimeAmount]*StatsSave),
	}
	this.getChan <- greq
	return <- greq.resultChan
}

//starts the event loop.
//this is necessary for it to do anything!
func (this *Stats) Start() {
	go this.eventLoop()
}

func (this *Stats) Close() error {
	//TODO: cleanly exit
	return nil
}

func (this *Stats) eventLoop() {
	//TODO: add kill chan
	for {
		select {
		case item := <- this.itemChan:
			this.add(item)
		case greq := <- this.getChan:
			greq.resultChan <- this.get()

		}
	}
}

// Gets a clone of the current state
func (this *Stats) get() map[timeamount.TimeAmount]*StatsSave{
	mp := make(map[timeamount.TimeAmount]*StatsSave)
	for ta,ss := range(this.items) {
		res := &StatsSave{
			Values : ss.Values.Clone(),
			TimeAmount: ss.TimeAmount,
			Epoch : ss.Epoch,
		}
		mp[ta] = res
	}
	return mp
}

func (this *Stats) add(item statsItem) {
	for ta, sts := range(this.items) {
		epoch := ta.ToTrendrrEpoch(time.Now())

		if epoch != sts.Epoch {
			// need to persist this.. 
			this.persist(sts)
			sts = &StatsSave{
				Epoch : epoch,
				TimeAmount : ta,
				Values : dynmap.New(),
			}
			this.items[ta] = sts
		}
		
		if item.typ == INC {
			val := sts.Values.MustInt64(item.Key, int64(0))
			sts.Values.PutWithDot(item.Key, int64(val+item.Val))	
		} else if item.typ == SET {
			sts.Values.PutWithDot(item.Key, int64(item.Val))
		}

		
	}
}

func (this *Stats) persist(item *StatsSave) {
	//Do something 
	json, _ := item.Values.MarshalJSON()

	log.Printf("TODO PERSISTING %s %s", item.TimeAmount.String(), string(json))
}