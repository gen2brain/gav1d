package av1

import (
	"bufio"
	"compress/gzip"
	"io"
	"os"
	"testing"
)

func openFixture(t *testing.T, name string) (*bufio.Scanner, func()) {
	t.Helper()

	f, err := os.Open("testdata/" + name + ".gz")
	if err != nil {
		t.Fatal(err)
	}

	zr, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		t.Fatal(err)
	}

	sc := bufio.NewScanner(zr)
	sc.Buffer(make([]byte, 0, 1<<20), 1<<25)

	return sc, func() {
		zr.Close()
		f.Close()
	}
}

func readFixture(t *testing.T, name string) []byte {
	t.Helper()

	f, err := os.Open("testdata/" + name + ".gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer zr.Close()

	data, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}

	return data
}
