//  Timeamount is a standard way to talk about a discrete amount of time.  
// it also has a convient encoding between an int and a time where # of timeamounts
// since trendrr epoch (Fri Dec 31 21:00:00 PST 1999)
//
// for now it only handles millis, seconds, minutes, hours, days.

package timeamount

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var TrendrrEpoch, ok = time.Parse("Mon Jan 2 15:04:05 MST 2006", "Fri Dec 31 21:00:00 PST 1999")
var USEast, err = time.LoadLocation("America/New_York")

type Timeframe string

const (
	MILLISECONDS Timeframe = "milli"
	SECONDS                = "s"
	MINUTES                = "min"
	HOURS                  = "hr"
	DAYS                   = "d" //days is not totally correct as it does not handle DST atm.
	//intentionally skip weeks ect (for now..)
	// WEEKS = "w"
	// MONTHS = "mo"
	// YEARS = "y"
)

type TimeAmount struct {
	amount    int
	timeframe Timeframe
}

// wraps a regular Time struct to contain the TimeAmount and associated Epoch.
type Time struct {
	time.Time
	TimeAmount   TimeAmount
	TrendrrEpoch int64
}

//checks if the given time is in NYC daylight savings time.
func IsEasternDayLight(time time.Time) bool {
	t := time.In(USEast)
	zone, _ := t.Zone()
	if zone == "EDT" {
		return true
	}
	return false
}

// Creates a new time object 
func NewTime(timeamount TimeAmount, time time.Time) Time {
	t := &Time{time, timeamount, timeamount.ToTrendrrEpoch(time)}
	return *t
}

// Create a new timeamount
func New(amount int, timeframe Timeframe) TimeAmount {
	ta := &TimeAmount{amount: amount, timeframe: timeframe}
	return *ta
}

// parses a timeamount usually in the form 15 minutes
func Parse(timeamount string) (TimeAmount, error) {
	numreg := regexp.MustCompile("[0-9]+")

	ta := &TimeAmount{}
	amount, err := strconv.Atoi(numreg.FindString(timeamount))
	if err != nil {
		return *ta, fmt.Errorf("Unable to parse TimeAmount: %s, caused by: %s", timeamount, err)
	}
	ta.amount = amount
	stringreg := regexp.MustCompile("[a-zA-Z]+")
	str := stringreg.FindString(timeamount)
	str = strings.ToLower(str)

	if strings.HasPrefix(str, "mil") {
		ta.timeframe = MILLISECONDS
		return *ta, nil
	} else if strings.HasPrefix(str, "s") {
		ta.timeframe = SECONDS
		return *ta, nil
	} else if strings.HasPrefix(str, "min") {
		ta.timeframe = MINUTES
		return *ta, nil
	} else if strings.HasPrefix(str, "h") {
		ta.timeframe = HOURS
		return *ta, nil
	} else if strings.HasPrefix(str, "d") {
		ta.timeframe = DAYS
		return *ta, nil
	} /*else if strings.HasPrefix(str, "mo"){
	      ta.timeframe = MONTHS
	      return *ta, true
	  } else if strings.HasPrefix(str, "y"){
	      ta.timeframe = YEARS
	      return *ta, true
	  } */
	return *ta, fmt.Errorf("Unable to parse TimeAmount: %s", timeamount)
}

func (this *TimeAmount) ToTrendrrEpoch(time time.Time) int64 {
	dur := time.Sub(TrendrrEpoch)
	epoch := int64(0)

	switch this.timeframe {
	case MILLISECONDS:
		epoch = int64(dur.Nanoseconds() / 1000000)
	case SECONDS:
		epoch = int64(dur.Seconds())
	case MINUTES:
		epoch = int64(dur.Minutes())
	case HOURS:
		epoch = int64(dur.Hours())
	case DAYS:
		//TODO: need to add 1 if time is in daylight savings time in NYC!
		epoch = int64(dur.Hours() / 24)
	}
	return int64(epoch / int64(this.amount))
}

//Translates a go time to a timeamount.Time
func (this *TimeAmount) ToTime(time time.Time) Time {
	epoch := this.ToTrendrrEpoch(time)
	return *&Time{time, *this, epoch}
}

func (this *TimeAmount) FromTrendrrEpoch(epoch int64) Time {
	str := ""
	amount := this.amount
	switch this.timeframe {
	case MILLISECONDS:
		str = "ms"
	case SECONDS:
		str = "s"
	case MINUTES:
		str = "m"
	case HOURS:
		str = "h"
	case DAYS:
		str = "h"
		amount = this.amount * 24
	}
	dur, err := time.ParseDuration(fmt.Sprintf("%d%s", (int64(amount) * epoch), str))
	if err != nil {
		log.Println(err)

	}
	t := TrendrrEpoch.Add(dur)
	return *&Time{t, *this, epoch}

}

// Translates to a regular duration
func (this *TimeAmount) ToDuration() (time.Duration, error) {
	str := ""
	amount := this.amount
	switch this.timeframe {
	case MILLISECONDS:
		str = "ms"
	case SECONDS:
		str = "s"
	case MINUTES:
		str = "m"
	case HOURS:
		str = "h"
	case DAYS:
		str = "h"
		amount = this.amount * 24
	}

	dur, err := time.ParseDuration(fmt.Sprintf("%d%s", amount, str))
	return dur, err
}

func (this *TimeAmount) String() string {
	return fmt.Sprintf("%d%s", this.amount, this.timeframe)
}
