package benchmark

import (
	"fmt"
	"time"
)

func slowFunc(ms int) (string, error) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
	return fmt.Sprintf("result %d", ms), nil
}
