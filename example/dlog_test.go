package dlog

import (
	"fmt"
	"mygolang/dlog"
	"testing"
)

func TestDlog(t *testing.T) {

	err := dlog.Init("./dlog.log")
	if err != nil {
		fmt.Println(err)
		return
	}

	dlog.Info("reqprice:", dlog.String("key", "apple"), dlog.Int("price", 100))
	dlog.Close()
}

func _BenchmarkCallerName(b *testing.B) {

	for i := 0; i < b.N; i++ {
		dlog.CallerName()
	}
}

func _BenchmarkCallerName1(b *testing.B) {

	for i := 0; i < b.N; i++ {
		dlog.CallerName1()
	}
}

func BenchmarkDlog(b *testing.B) {

	err := dlog.Init("./dlog.log")
	if err != nil {
		fmt.Println(err)
		return
	}
	// defer dlog.Close()

	for i := 0; i < b.N; i++ {
		dlog.Info("reqprice:", dlog.String("key", "apple"), dlog.Int("price", 100))
	}
}
