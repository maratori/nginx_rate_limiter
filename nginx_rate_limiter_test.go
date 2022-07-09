package main_test

import (
	"context"
	"fmt"
	"github.com/lithammer/dedent"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"
)

func TestSimple(t *testing.T) {
	endpoint := StartNginx(t, `
		limit_req_zone $request_uri zone=my_zone:1m rate=30r/m;
		server {
			listen 80;
			location / {
				limit_req zone=my_zone;
				try_files $uri /index.html;
			}
		}
	`)

	Print(GetMany(endpoint, map[string]int{
		"/0": 2,
		"/1": 2,
	})...)

	println()
	time.Sleep(time.Second)

	Print(GetMany(endpoint, map[string]int{
		"/0": 2,
		"/1": 2,
	})...)

	println()
	time.Sleep(time.Second)

	Print(GetMany(endpoint, map[string]int{
		"/0": 2,
		"/1": 2,
	})...)
}

func TestSimpleBurst(t *testing.T) {
	endpoint := StartNginx(t, `
		limit_req_zone $request_uri zone=my_zone:1m rate=12r/m;
		server {
			listen 80;
			location / {
				limit_req zone=my_zone burst=2;
				try_files $uri /index.html;
			}
		}
	`)

	Print(GetMany(endpoint, map[string]int{
		"/0": 5,
		"/1": 5,
	})...)
}

func TestSimpleBurstNodelay(t *testing.T) {
	endpoint := StartNginx(t, `
		limit_req_zone $request_uri zone=my_zone:1m rate=30r/m;
		server {
			listen 80;
			location / {
				limit_req zone=my_zone burst=2 nodelay;
				try_files $uri /index.html;
			}
		}
	`)

	Print(GetMany(endpoint, map[string]int{
		"/0": 5,
		"/1": 5,
	})...)

	println()
	time.Sleep(time.Second)

	Print(GetMany(endpoint, map[string]int{
		"/0": 2,
		"/1": 2,
	})...)
}

func TestBypassRateLimiterSmallZoneSize(t *testing.T) {
	endpoint := StartNginx(t, `
		limit_req_zone $huge$request_uri zone=my_zone:32k rate=1r/m;
		server {
			listen 80;
			location / {
				set $x 1234567890;
				set $y $x$x$x$x$x$x$x$x$x$x;
				set $z $y$y$y$y$y$y$y$y$y$y;
				set $huge $z$z$z$z$z;
				limit_req zone=my_zone;
				try_files $uri /index.html;
			}
		}
	`)

	Print(GetMany(endpoint, map[string]int{
		"/some": 2,
		"/any":  2,
	})...)
	println()
	for _, x := range "123451234512345" {
		Print(Get(endpoint, "/"+string(x)))
	}
}

func TestBypassRateLimiterSmallZoneSizeBurst(t *testing.T) {
	endpoint := StartNginx(t, `
		limit_req_zone $huge$request_uri zone=my_zone:32k rate=12r/m;
		server {
			listen 80;
			location / {
				set $x 1234567890;
				set $y $x$x$x$x$x$x$x$x$x$x;
				set $z $y$y$y$y$y$y$y$y$y$y;
				set $huge $z$z$z$z$z;
				limit_req zone=my_zone burst=2;
				try_files $uri /index.html;
			}
		}
	`)

	const (
		original = "/x"
		nFirst   = 5
		nSecond  = 5
		other    = "12345"
		total    = nFirst + len(other) + nSecond
	)

	ch := make(chan Result, total)
	go func() {
		res := GetMany(endpoint, map[string]int{
			original: nFirst,
		})
		for _, r := range res {
			ch <- r
		}
	}()

	time.Sleep(100 * time.Millisecond)

	for _, x := range other {
		go func(x string) {
			ch <- Get(endpoint, "/"+x)
		}(string(x))
	}

	time.Sleep(100 * time.Millisecond)

	go func() {
		res := GetMany(endpoint, map[string]int{
			original: nSecond,
		})
		for _, r := range res {
			ch <- r
		}
	}()

	res := make([]Result, total)
	for i := range res {
		res[i] = <-ch
	}
	Print(res...)
}

func Print(res ...Result) {
	sort.Slice(res, func(i, j int) bool {
		a := res[i]
		b := res[j]
		sa := a.Start.Round(10 * time.Millisecond)
		sb := b.Start.Round(10 * time.Millisecond)
		if sa.Equal(sb) {
			da := a.Duration.Round(time.Millisecond)
			db := b.Duration.Round(time.Millisecond)
			if da == db {
				if a.Status == b.Status {
					return a.URL < b.URL
				}
				return a.Status < b.Status
			}
			return da < db
		}
		return sa.Before(sb)
	})
	for _, r := range res {
		status := "❌"
		if r.Status == http.StatusOK {
			status = "✅"
		}
		mark := ""
		if r.Duration > time.Second {
			mark = "*"
		}
		fmt.Printf("%s   %-15s %-10s %20s %s\n", status, r.URL, r.Start.Format("04:05.999"), r.Duration, mark)
	}
}

func PrintHeader() {
	fmt.Printf("%s   %-15s %-10s %20s %s\n", "  ", "URL", "Start time", "Duration", "")
}

type Result struct {
	URL      string
	Status   int
	Start    time.Time
	Duration time.Duration
}

func Get(base string, url string) Result {
	start := time.Now()
	resp, err := http.Get(base + url)
	end := time.Now().Sub(start)
	if err != nil {
		panic(err)
	}
	io.Copy(io.Discard, resp.Body)
	defer resp.Body.Close()
	return Result{
		URL:      url,
		Status:   resp.StatusCode,
		Start:    start,
		Duration: end,
	}
}

func GetMany(base string, urls map[string]int) []Result {
	ch := make(chan Result)
	total := 0
	for url, n := range urls {
		total += n
		for range make([]struct{}, n) {
			go func(url string) {
				ch <- Get(base, url)
			}(url)
		}
	}
	res := make([]Result, total)
	for i := range res {
		res[i] = <-ch
	}
	close(ch)
	return res
}

func StartNginx(t *testing.T, conf string) string {
	fmt.Println("# " + t.Name())
	fmt.Println(conf)

	const dockerfile = `
		FROM nginx:alpine
		COPY default.conf /etc/nginx/conf.d/default.conf
		COPY index.html /etc/nginx/html/index.html
	`

	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(dedent.Dedent(dockerfile)), 0666)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "default.conf"), []byte(dedent.Dedent(conf)), 0666)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "index.html"), []byte("ok"), 0666)
	require.NoError(t, err)

	ctx := context.Background()
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context: dir,
			},
			ExposedPorts: []string{"80/tcp"},
			WaitingFor:   wait.ForListeningPort("80"),
			AutoRemove:   true,
		},
		Started: true,
		Logger:  log.New(io.Discard, "", 0),
	})
	require.NoError(t, err)
	t.Cleanup(func() { c.Terminate(ctx) })

	endpoint, err := c.Endpoint(ctx, "http")
	require.NoError(t, err)

	println()
	t.Cleanup(func() { println() })

	PrintHeader()

	return endpoint
}
