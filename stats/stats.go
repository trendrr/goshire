package aq

import(
	"github.com/trendrr/goshire/dynmap"
	"github.com/trendrr/goshire/timeamount"
	"time"

)

// A simple stats collector

type Stats struct {
	itemChan chan statsItem
	Persister StatsPersister
	items map[timeamount.TimeAmount]&StatsSave

}

type StatsPersister interface {
	Persist(t timeamount.TimeAmount, val dynmap.DynMap )

}

type statsItem struct {
	Key string
	Val int64
}

type StatsSave struct {
	Epoch int64
	TimeAmount timeamount.TimeAmount
	Values dynmap.DynMap
}

// Creates a new Stats tracker.  and starts the eventLoop.
// timeamounts should be in the form "{num} {timeframe}" 
// example 
// NewStats("1 minute", "30 minute", "1 day")
func NewStats(timeamounts ...string) (*Stats, error) {
	s := &Stats{
		itemChan : make(chan statsItem, 500),
		items : make(map[timeamount.TimeAmount]&StatsSave),
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
	go s.eventLoop()
}

func (this *Stats) Inc(key string, val int64) {
	select {
	case this.itemChan <- statsItem{Key : key, Val : val} :
	default :
		log.Printf( "Could not Inc Key: %s Val: %d", key, val)
	}
}

func (this *Stats) eventLoop() {
	//TODO: add kill chan
	for {
		select {
		case item := <- this.itemChan:
			this.add(item)
		}
	}
}


func (this *Stats) add(item StatsItem) {
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
		val := sts.Values.MustInt64(item.Key, int64(0))
		sts.Values.PutWithDot(item.Key, int64(val+item.Val))
	}
}

func (this *Stats) persist(item StatsSave) {
	//Do something 
	json, err := item.MarshalJSON()

	log.Printf("TODO PERSISTING %s %s", item.TimeAmount.String(), string(json))
}