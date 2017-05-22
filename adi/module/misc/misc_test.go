package misc

import (
	"fmt"
	"testing"
	"time"
)

func TestTimeStamp(t *testing.T) {
	const a_timestamp = "1401383885.000061"
	ts, err := parseTimestamp(a_timestamp)
	if err != nil {
		t.Error(err)
	}
	mt := time.Unix(ts.Unix, 0)
	diff := time.Since(mt)
	fmt.Println(diff, ts)
}
