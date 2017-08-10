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

	"github.com/avahowell/reqlimit"
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
BenchmarkLimitedHandlerSingleIP-4    5000000       383 ns/op     120 B/op       3 allocs/op
BenchmarkLimitedHandlerManyIPs-4     2000000       532 ns/op     203 B/op       2 allocs/op
BenchmarkRawHandler-4              100000000      13.3 ns/op       0 B/op       0 allocs/op
```

