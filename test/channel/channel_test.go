package channel

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/samber/lo"
)

func TestTimeAfterAsChannel(t *testing.T) {
	c := make(chan int)
	go func(c chan int) {
		time.Sleep(time.Second)
		c <- 1
	}(c)
	select {
	case <-time.After(time.Second * 100):
		println(1)
	case <-c:
		println(2)
	default:
		println(3)
	}
}

func TestGc(t *testing.T) {
	go func() {
		for {
		}
	}()
	time.Sleep(time.Millisecond)
	runtime.GC()
	println("OK")
}

func TestContext(t *testing.T) {
	ctx, cancel := context.WithDeadline(context.TODO(), time.Now().AddDate(0, 0, 1))
	defer cancel()
	select {
	case <-ctx.Done():
		println(1)
	}
}

type Person struct {
	age int
}

func (p Person) getAge() int {
	return p.age
}
func (p *Person) setAge() {
	p.age++
}

func TestCommon(t *testing.T) {
	handlerEmailPrefix := []string{"test1", "test2"}
	str := strings.Join(lo.Map(handlerEmailPrefix, func(item string, _ int) string { return fmt.Sprintf(`<at email=%s@bambulab.com></at>`, item) }), "")
	println(str)
}

func TestMap(t *testing.T) {
	c1 := make(chan int)
	c2 := make(chan int)
	m := make(map[chan int]int)
	m[c1] = 1
	m[c2] = 2

	for key, val := range m {
		println("key = ", key, " val = ", val)
	}
}

func TestSlice(t *testing.T) {
	s := []int{1, 2, 3, 4}
	s = append(s, 6, 7)
}

type Data struct {
	message string
}

func TestNilPointer(t *testing.T) {
	var s *Data
	s.message = "hello"
	println(s.message)
}
