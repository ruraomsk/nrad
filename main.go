package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"
)

var mutex sync.Mutex

type stateRadar struct {
	name     string
	state    bool
	start    time.Time
	duration time.Duration
	count    int
}

func (s *stateRadar) change(state bool, t time.Time) {
	if state == s.state {
		if state {
			s.count++
		} else {
			s.count = 0
		}
		return
	}
	s.state = state
	if !state {
		//Машина уехала
		s.duration = t.Sub(s.start)
		s.count = 0
		// fmt.Printf("%s %s \n", t.String(), s.start.String())
	} else {
		if s.count == 0 {
			s.start = t
		}
		s.count++
	}
}
func (s *stateRadar) getDuration() time.Duration {
	if s.count == 0 {
		return s.duration
	}
	return time.Since(s.start)
}

func (s *stateRadar) toString() string {
	return fmt.Sprintf("%s\t%v\t%03d\t%s\t%v", s.name, s.state, s.count, s.start.String(), s.getDuration())
}

var chanelCount = 5
var chs map[string]stateRadar

func main() {
	go writerRadar(15000)
	time.Sleep(time.Second)
	go readerRadar("127.0.0.1", 15000)
	fmt.Println("All started...")
	for {
		time.Sleep(5 * time.Second)
		mutex.Lock()
		fmt.Println("===================================================================")
		for i := 0; i < chanelCount; i++ {
			v := chs[nameFromNumber(i)]
			fmt.Println(v.toString())
		}
		mutex.Unlock()
	}
}
func writerRadar(port int) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		fmt.Println(err)
		return
	}
	for {
		socket, err := ln.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}
		fmt.Println(socket.RemoteAddr().String())
		go senderMessages(socket)
	}
}
func nameFromNumber(i int) string {
	return fmt.Sprintf("%dC4", i+1)
}
func senderMessages(socket net.Conn) {
	writer := bufio.NewWriter(socket)
	defer socket.Close()
	name := ""
	for {
		for i := 0; i < chanelCount; i++ {
			time.Sleep(50 * time.Millisecond)
			name = nameFromNumber(i)
			if rand.Int()%2 == 0 {
				status := rand.Int() % 3
				ts := time.Now().Unix()
				ns := time.Now().Nanosecond()
				switch status {
				case 0:
					writer.WriteString(fmt.Sprintf("%s %d.%d\r\n", name, ts, ns))
				case 1:
					writer.WriteString(fmt.Sprintf("%s %d.%d %d\r\n", name, ts, ns, status))
				case 2:
					writer.WriteString(fmt.Sprintf("%s %d.%d %d\r\n", name, ts, ns, status))
				}
				err := writer.Flush()
				if err != nil {
					fmt.Println(err)
					break
				}

			}
		}

	}
}
func readerRadar(host string, port int) {
	chs = make(map[string]stateRadar)
	for {
		socket, err := net.Dial("tcp", fmt.Sprintf("%s:%d", host, port))
		if err != nil {
			fmt.Println(err)
			return
		}
		reader := bufio.NewReader(socket)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				fmt.Println(err)
				break
			}
			line = strings.ReplaceAll(line, "\r", "")
			line = strings.ReplaceAll(line, "\n", "")
			// fmt.Printf("->%s\n", line)
			name, s, t := unpack(line)
			if s == 2 {
				continue
			}
			mutex.Lock()
			st, is := chs[name]
			if !is {
				st = stateRadar{name: name, state: false, count: 0, duration: 0}
			}
			if s == 0 {
				st.change(false, t)
			} else {
				st.change(true, t)
			}
			chs[name] = st
			mutex.Unlock()
		}
		socket.Close()
	}

}
func unpack(l string) (name string, state int, t time.Time) {
	ls := strings.Split(l, " ")
	if len(ls) < 2 || len(ls) > 3 {
		fmt.Printf(" [%s]\n", l)
		return "", 2, time.Now()
	}

	name = ls[0]
	var ts int64
	var tn int64
	tss := strings.Split(ls[1], ".")
	_, _ = fmt.Sscan(tss[0], &ts)
	_, _ = fmt.Sscan(tss[1], &tn)
	t = time.Unix(ts, tn)
	// fmt.Println(time.Since(t))
	if len(ls) == 2 {
		return name, 0, t
	}
	if ls[2] == "1" {
		return name, 1, t
	}
	if ls[2] == "2" {
		return name, 2, t
	}
	return name, 2, t
}
