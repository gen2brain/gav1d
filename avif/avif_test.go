package avif

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gen2brain/gav1d/av1"
)

const avifdec = "/temp/av1/libavif-build/avifdec"

type y4m struct {
	w, h   int
	depth  int
	layout int
	alpha  bool
	planes [4][]byte
	stride [4]int
}

// readY4M parses the single frame avifdec writes.
func readY4M(b []byte) (*y4m, error) {
	i := bytes.IndexByte(b, '\n')
	if i < 0 {
		return nil, fmt.Errorf("no y4m header")
	}
	hdr := string(b[:i])
	rest := b[i+1:]

	y := &y4m{depth: 8, layout: av1.LayoutI420}
	for _, f := range strings.Fields(hdr)[1:] {
		v := f[1:]
		switch f[0] {
		case 'W':
			y.w, _ = strconv.Atoi(v)
		case 'H':
			y.h, _ = strconv.Atoi(v)
		case 'C':
			cs := v
			if i := strings.LastIndexByte(cs, 'p'); i >= 0 {
				if d, err := strconv.Atoi(cs[i+1:]); err == nil {
					y.depth = d
					cs = cs[:i]
				}
			}
			if rest, ok := strings.CutSuffix(cs, "alpha"); ok {
				y.alpha = true
				cs = rest
			}
			switch {
			case strings.HasPrefix(cs, "mono"):
				y.layout = av1.LayoutI400
				if d, err := strconv.Atoi(cs[len("mono"):]); err == nil {
					y.depth = d
				}
			case strings.HasPrefix(cs, "420"):
				y.layout = av1.LayoutI420
			case strings.HasPrefix(cs, "422"):
				y.layout = av1.LayoutI422
			case strings.HasPrefix(cs, "444"):
				y.layout = av1.LayoutI444
			default:
				return nil, fmt.Errorf("y4m color space %q", v)
			}
		}
	}

	i = bytes.IndexByte(rest, '\n')
	if i < 0 || !bytes.HasPrefix(rest, []byte("FRAME")) {
		return nil, fmt.Errorf("no y4m frame")
	}
	rest = rest[i+1:]

	bpp := 1
	if y.depth > 8 {
		bpp = 2
	}
	ssHor, ssVer := subsampling(y.layout)

	for pl := range 4 {
		if pl != 0 && y.layout == av1.LayoutI400 {
			break
		}
		if pl == 3 && !y.alpha {
			break
		}
		pw, ph := y.w, y.h
		if pl != 0 && pl != 3 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		n := pw * ph * bpp
		if len(rest) < n {
			return nil, fmt.Errorf("y4m plane %d truncated", pl)
		}
		y.planes[pl] = rest[:n]
		y.stride[pl] = pw * bpp
		rest = rest[n:]
	}

	return y, nil
}

func decodeRef(t *testing.T, path string) *y4m {
	t.Helper()

	if _, err := os.Stat(avifdec); err != nil {
		t.Skip("avifdec not built")
	}
	out := filepath.Join(t.TempDir(), "ref.y4m")
	cmd := exec.Command(avifdec, "--no-strict", path, out)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("avifdec: %v\n%s", err, b)
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	y, err := readY4M(b)
	if err != nil {
		t.Fatal(err)
	}

	return y
}

func comparePlanes(t *testing.T, pic *av1.Picture, ref *y4m) {
	t.Helper()

	if pic.Width != ref.w || pic.Height != ref.h {
		t.Fatalf("size %dx%d, want %dx%d", pic.Width, pic.Height, ref.w, ref.h)
	}
	if pic.BitDepth != ref.depth {
		t.Fatalf("depth %d, want %d", pic.BitDepth, ref.depth)
	}
	if pic.Layout != ref.layout {
		t.Fatalf("layout %d, want %d", pic.Layout, ref.layout)
	}

	bpp := 1
	if ref.depth > 8 {
		bpp = 2
	}
	ssHor, ssVer := subsampling(ref.layout)

	for pl := range 3 {
		if ref.planes[pl] == nil {
			continue
		}
		pw, ph := ref.w, ref.h
		if pl != 0 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		for y := range ph {
			got := pic.Data[pl][y*pic.Stride[pl]:][:pw*bpp]
			want := ref.planes[pl][y*ref.stride[pl]:][:pw*bpp]
			if !bytes.Equal(got, want) {
				for x := range pw * bpp {
					if got[x] != want[x] {
						t.Fatalf("plane %d row %d byte %d: %d, want %d",
							pl, y, x, got[x], want[x])
					}
				}
			}
		}
	}
}

func avifFiles(t *testing.T, dir string) []string {
	t.Helper()

	var out []string
	err := filepath.Walk(dir, func(p string, fi os.FileInfo, err error) error {
		if err == nil && !fi.IsDir() && strings.HasSuffix(p, ".avif") {
			out = append(out, p)
		}

		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(out)

	return out
}

// TestConformance decodes an external AVIF corpus, comparing every plane
// against avifdec. Set AVIF_CONFORMANCE_DIR (colon separated) to run it.
func TestConformance(t *testing.T) {
	env := os.Getenv("AVIF_CONFORMANCE_DIR")
	if env == "" {
		t.Skip("set AVIF_CONFORMANCE_DIR")
	}
	dirs := strings.Split(env, ":")

	var exact, skipped int
	fails := map[string]int{}

	for _, dir := range dirs {
		if _, err := os.Stat(dir); err != nil {
			t.Skip("conformance corpus not present")
		}
		for _, path := range avifFiles(t, dir) {
			name := strings.TrimPrefix(path, dir+"/")
			t.Run(name, func(t *testing.T) {
				ref := decodeRef(t, path)

				data, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				f, err := parse(data)
				if err != nil {
					fails[err.Error()]++
					t.Fatalf("parse: %v", err)
				}
				pic, _, err := f.decodePrimary()
				if err != nil {
					fails[err.Error()]++
					t.Fatalf("decode: %v", err)
				}
				comparePlanes(t, pic, ref)
				exact++
			})
		}
	}

	t.Logf("%d bit-exact, %d skipped", exact, skipped)
	keys := make([]string, 0, len(fails))
	for k := range fails {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return fails[keys[i]] > fails[keys[j]] })
	for _, k := range keys {
		t.Logf("%4d  %s", fails[k], k)
	}
}

func planeMD5(pic *av1.Picture) string {
	h := md5.New()
	ssHor, ssVer := subsampling(pic.Layout)
	bpp := 1
	if pic.BitDepth > 8 {
		bpp = 2
	}

	for pl := range 3 {
		if pic.Data[pl] == nil {
			continue
		}
		pw, ph := pic.Width, pic.Height
		if pl != 0 {
			pw, ph = (pw+ssHor)>>ssHor, (ph+ssVer)>>ssVer
		}
		for y := range ph {
			h.Write(pic.Data[pl][y*pic.Stride[pl]:][:pw*bpp])
		}
	}

	return hex.EncodeToString(h.Sum(nil))
}

// TestDecode compares the bundled files against the plane MD5s in md5.txt.
// Set AVIF_UPDATE_MD5 to regenerate them, which needs avifdec.
func TestDecode(t *testing.T) {
	const dir = "testdata"

	files, err := filepath.Glob(filepath.Join(dir, "*.avif"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	if len(files) == 0 {
		t.Fatal("no test files")
	}

	if os.Getenv("AVIF_UPDATE_MD5") != "" {
		var b strings.Builder
		for _, path := range files {
			ref := decodeRef(t, path)
			h := md5.New()
			for pl := range 3 {
				if ref.planes[pl] != nil {
					h.Write(ref.planes[pl])
				}
			}
			fmt.Fprintf(&b, "%s %s\n", filepath.Base(path),
				hex.EncodeToString(h.Sum(nil)))
		}
		if err := os.WriteFile(filepath.Join(dir, "md5.txt"), []byte(b.String()), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote md5.txt for %d files", len(files))

		return
	}

	want := map[string]string{}
	data, err := os.ReadFile(filepath.Join(dir, "md5.txt"))
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		if f := strings.Fields(line); len(f) == 2 {
			want[f[0]] = f[1]
		}
	}

	for _, path := range files {
		name := filepath.Base(path)
		t.Run(name, func(t *testing.T) {
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			f, err := parse(raw)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			pic, alpha, err := f.decodePrimary()
			if err != nil {
				t.Fatalf("decode: %v", err)
			}
			got := planeMD5(pic)
			if want[name] == "" {
				t.Fatalf("no md5 for %s", name)
			}
			if got != want[name] {
				t.Fatalf("md5 = %s, want %s", got, want[name])
			}
			if strings.HasPrefix(name, "alpha-") && alpha == nil {
				t.Fatal("alpha item not found")
			}
		})
	}
}
