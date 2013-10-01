package closer


// Simple package to handle clean shut down behavior
import (
	"io"
	"sync"
	"os"
	"os/signal"
	"syscall"
	"log"
)


type cl struct {
	items []io.Closer
	once sync.Once
}
var c = cl{
	items : make([]io.Closer, 0),
}

func Register(closer io.Closer) {
	c.Register(closer)
}

//start listening
func (this *cl) listen() {
	//listen for kill sig
	killchan := make(chan os.Signal, 2)
	signal.Notify(killchan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<- killchan
		log.Println("Attempting clean shutdown")
		for _, item := range(this.items) {
			log.Println("Closing ", item)
			item.Close()
		}
		log.Println("GOOD BYE")
		os.Exit(0)
	}()
}

func (this *cl) Register(closer io.Closer) {
	this.once.Do(this.listen)
	this.items = append(this.items, closer)
}

