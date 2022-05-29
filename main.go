package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/sync/errgroup"
)

func serveApp(ctx context.Context, stop <-chan struct{}) error {
	mux := http.NewServeMux()
	s := http.Server{Addr: ":8080", Handler: mux}

	mux.HandleFunc("/", func(resp http.ResponseWriter, req *http.Request) {
		fmt.Fprintln(resp, "Hello World 1")
	})

	go func() {
		<-stop
		s.Shutdown(context.Background())
	}()

	return s.ListenAndServe()
}

func serveDebug(ctx context.Context, stop <-chan struct{}) error {
	mux := http.DefaultServeMux
	s := http.Server{Addr: ":8081", Handler: mux}

	go func() {
		<-stop
		s.Shutdown(context.Background())
	}()

	return s.ListenAndServe()
}

func main() {
	var stopped bool = false
	done := make(chan error, 2)
	stop := make(chan struct{})

	WebApp := func(ctx context.Context) {
		g, ctx := errgroup.WithContext(ctx)
		g.Go(func() error {
			done <- serveDebug(ctx, stop)
			return nil
		})
		g.Go(func() error {
			done <- serveApp(ctx, stop)
			return nil
		})

		g.Go(func() error {
			c := make(chan os.Signal)
			signal.Notify(c, os.Interrupt, os.Kill, syscall.SIGTERM, syscall.SIGUSR1, syscall.SIGUSR2)
			fmt.Println("启动")
			s := <-c
			fmt.Println("退出信号", s)
			close(stop)
			stopped = true
			return nil
		})

		if err := g.Wait(); err != nil {
			fmt.Println(err)
		}
	}

	WebApp(context.Background())

	for i := 0; i < cap(done); i++ {
		if err := <-done; err != nil {
			fmt.Printf("error %v \n", err)
		}
		if !stopped {
			stopped = true
			close(stop)
		}
	}

}
