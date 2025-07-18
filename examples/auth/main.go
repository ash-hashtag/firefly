package main

import (
	"fmt"
	"log"
	"sync"
	"time"
)

func ExampleFetchWithMutexLock(mu *sync.RWMutex) {

	start := time.Now()
	mu.Lock()
	end := time.Now()
	defer mu.Unlock()

	log.Printf("Started at %v and acquired WRITE lock at %v, time took: %d ms", start, end, end.UnixMilli()-start.UnixMilli())
	time.Sleep(time.Second * 5)

}

func ExampleReadWithMutexLock(mu *sync.RWMutex) {
	start := time.Now()
	end := time.Now()
	mu.RLock()
	defer mu.RUnlock()
	log.Printf("Started at %v and acquired READ lock at %v, time took: %d ms", start, end, end.UnixMilli()-start.UnixMilli())
	time.Sleep(time.Second * 3)
}

func ForeverLoop(interval time.Duration) {
	var count = 0
	for {
		log.Printf("%d", count)
		time.Sleep(interval)
		count += 1
	}

}

func ChannelTest(c chan *[]byte) {
	for value := range c {
		log.Printf("Received value from channel %s %p", *value, value)
	}
}

func MutexTest() {
	mu := sync.RWMutex{}

	go ForeverLoop(time.Second)
	go ExampleReadWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
	go ExampleFetchWithMutexLock(&mu)
	go ExampleFetchWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
	go ExampleReadWithMutexLock(&mu)
}

func TestChannelComparison() {
	ch := make(chan string)

	ch2 := ch

	go func() {
		for value := range ch2 {
			if ch2 != ch {
				log.Fatalf("Both channels are not equal")
			}
			log.Println(value)
		}

	}()

	for i := 0; i < 1000; i++ {
		go func() {
			ch <- fmt.Sprintf("%d", i)
		}()

		time.Sleep(time.Second * 1)
	}

}

func main() {
	// keys := utils.NewGooglePublicKeysHandler("lupyd-fb")
	// log.Printf("%v", keys.Keys)
	// c := make(chan *[]byte)

	// go ChannelTest(c)

	// for i := 0; i < 10; i++ {
	// 	value := []byte(strconv.Itoa(i))
	// 	address := &value
	// 	log.Printf("Sending %s with address %p", value, address)
	// 	c <- &value
	// 	time.Sleep(time.Second)
	// }
	//
	TestChannelComparison()

	time.Sleep(time.Second * 20)
}
