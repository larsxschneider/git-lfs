//go:generate goversioninfo -icon=script/windows-installer/git-lfs-logo.ico

package main

import (
	"fmt"
	"os"
	"io/ioutil"
	"os/signal"
	"runtime"
	"sync"
	"syscall"
	"time"
	"runtime/pprof"
	 "math/rand"

	"github.com/git-lfs/git-lfs/commands"
)

func main() {
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, os.Kill)

	var once sync.Once

	done := make(chan struct{})
	wg := new(sync.WaitGroup)
	wg.Add(1)

	s1 := rand.NewSource(time.Now().UnixNano())
	r1 := rand.New(s1)
	run := r1.Intn(100)


	// ### CPU PROFILING
	fcpu, err := ioutil.TempFile("/Users/lars/Temp/dump", fmt.Sprintf("cpu-%d-", run))
	if err != nil {
		os.Exit(1)
	}
	pprof.StartCPUProfile(fcpu)
	defer pprof.StopCPUProfile()


	// ### MEMORY PROFILING
	// TODO: doesn't work on macOS
	fmem, err := ioutil.TempFile("/Users/lars/Temp/dump", fmt.Sprintf("mem-%d-", run))
	if err != nil {
		panic(err.Error())
	}



	// ### MEORY STATS
	go func() {
		defer wg.Done()

		f, err := ioutil.TempFile("/Users/lars/Temp/dump", fmt.Sprintf("memstat-%d-", run))
		if err != nil {
			return
		}

		defer f.Close()

		for {
			select {
			case <-time.After(100 * time.Millisecond):
				var ms runtime.MemStats
				runtime.ReadMemStats(&ms)
				fmt.Fprintf(f, "%+v\n", ms)
			case <-done:
				return
			}
		}
	}()

	go func() {
		for {
			sig := <-c
			if err := pprof.WriteHeapProfile(fmem); err != nil {
				panic(err.Error())
			}
			fmem.Close()

			once.Do(commands.Cleanup)
			fmt.Fprintf(os.Stderr, "\nExiting because of %q signal.\n", sig)

			exitCode := 1
			if sysSig, ok := sig.(syscall.Signal); ok {
				exitCode = int(sysSig)
			}
			close(done)
			wg.Wait()
			os.Exit(exitCode + 128)
		}
	}()

	commands.Run()

	if err := pprof.WriteHeapProfile(fmem); err != nil {
		panic(err.Error())
	}
	fmem.Close()
	once.Do(commands.Cleanup)
}
