package av1

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

func flatten(v reflect.Value, dst []int64) []int64 {
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := range v.Len() {
			dst = flatten(v.Index(i), dst)
		}
	case reflect.Struct:
		for i := range v.NumField() {
			dst = flatten(v.Field(i), dst)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		dst = append(dst, v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		dst = append(dst, int64(v.Uint()))
	default:
		panic("unsupported kind " + v.Kind().String())
	}

	return dst
}

func TestTables(t *testing.T) {
	goTables := map[string]any{
		"dav1d_al_part_ctx":           alPartCtx,
		"dav1d_block_dimensions":      blockDimensions,
		"dav1d_cdef_directions":       cdefDirections,
		"dav1d_dq_tbl":                dqTbl,
		"dav1d_filter_dir":            filterDir,
		"dav1d_filter_mode_to_y_mode": filterModeToYMode,
		"dav1d_intra_mode_context":    intraModeContext,
		"dav1d_partition_type_count":  partitionTypeCount,
		"dav1d_skip_ctx":              skipCtx,
		"dav1d_tx_type_class":         txTypeClass,
		"dav1d_tx_types_per_set":      txTypesPerSet,
		"dav1d_wedge_ctx_lut":         wedgeCtxLut,
		"dav1d_ymode_size_context":    ymodeSizeContext,
	}

	for lvl := range 15 {
		for c := range 2 {
			for t2 := range nRectTxSizes {
				goTables[fmt.Sprintf("dav1d_qm_tbl[%d][%d][%d]", lvl, c, t2)] = qmTbl[lvl][c][t2]
			}
		}
	}

	for i := range nRectTxSizes {
		goTables[fmt.Sprintf("dav1d_scans[%d]", i)] = scans[i]
		goTables[fmt.Sprintf("dav1d_last_nonzero_col_from_eob[%d]", i)] = lastNonzeroColFromEob[i]
	}

	sc, closeFixture := openFixture(t, "tables.txt")
	defer closeFixture()

	seen := 0
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) == 0 || fields[0] != "T" {
			continue
		}

		name := fields[1]
		tbl, ok := goTables[name]
		if !ok {
			t.Errorf("%s: no Go equivalent", name)

			continue
		}

		n, err := strconv.Atoi(fields[2])
		if err != nil {
			t.Fatal(err)
		}
		if len(fields) != 3+n {
			t.Fatalf("%s: got %d values, want %d", name, len(fields)-3, n)
		}

		got := flatten(reflect.ValueOf(tbl), nil)
		if len(got) != n {
			t.Errorf("%s: Go table has %d elements, dav1d has %d", name, len(got), n)

			continue
		}

		for i := range n {
			want, err := strconv.ParseInt(fields[3+i], 10, 64)
			if err != nil {
				t.Fatal(err)
			}
			if got[i] != want {
				t.Errorf("%s[%d] = %d, want %d", name, i, got[i], want)

				break
			}
		}

		seen++
	}

	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}

	if seen != len(goTables) {
		t.Errorf("checked %d tables, have %d", seen, len(goTables))
	}
}
