package xmail

import (
    "bytes"
    "github.com/d3code/pkg/xerr"
    "os"
    "testing"
)

func TestParse(t *testing.T) {
    msg, err := os.ReadFile("_test/Announcement- Rules and Hooks End of Life.eml")
    xerr.ExitIfError(err)

    r2 := bytes.NewReader(msg)
    Parse(r2)
}
