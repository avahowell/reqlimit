# reqlimit

reqlimit is a tiny Golang middleware you can use to limit the frequency of requests to your server, on a per-IP basis.

## Usage Example

```go
package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/johnathanhowell/reqlimit"
)

func protectedHandler(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprintf(w, "you have successfully requested a protected resource!")
}

func main() {
	handler := http.HandlerFunc(protectedHandler)
	limitedHandler := reqlimit.New(handler, 10, time.Second * 5) // limit the handler to 10 requests every 5 seconds. 

	log.Fatal(http.ListenAndServe(":8080", limitedHandler) )
}
```

## Benchmarks
A limited handler is a bit slower and requires a bit more memory than a naked handler. Here's the benchmarks:

```
BenchmarkLimitedHandler-8         200000              5379 ns/op            2101 B/op         16 allocs/op
BenchmarkHandler-8               2000000               987 ns/op             656 B/op          6 allocs/op
```

