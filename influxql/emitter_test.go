package influxql_test

import (
	"testing"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/skia-dev/influxdb/influxql"
	"github.com/skia-dev/influxdb/models"
	"github.com/skia-dev/influxdb/pkg/deep"
)

// Ensure the emitter can group iterators together into rows.
func TestEmitter_Emit(t *testing.T) {
	// Build an emitter that pulls from two iterators.
	e := influxql.NewEmitter([]influxql.Iterator{
		&FloatIterator{Points: []influxql.FloatPoint{
			{Name: "cpu", Tags: ParseTags("region=west"), Time: 0, Value: 1},
			{Name: "cpu", Tags: ParseTags("region=west"), Time: 1, Value: 2},
		}},
		&FloatIterator{Points: []influxql.FloatPoint{
			{Name: "cpu", Tags: ParseTags("region=west"), Time: 1, Value: 4},
			{Name: "cpu", Tags: ParseTags("region=north"), Time: 0, Value: 4},
			{Name: "mem", Time: 4, Value: 5},
		}},
	}, true, 0)
	e.Columns = []string{"col1", "col2"}

	// Verify the cpu region=west is emitted first.
	if row := e.Emit(); !deep.Equal(row, &models.Row{
		Name:    "cpu",
		Tags:    map[string]string{"region": "west"},
		Columns: []string{"col1", "col2"},
		Values: [][]interface{}{
			{time.Unix(0, 0).UTC(), float64(1), nil},
			{time.Unix(0, 1).UTC(), float64(2), float64(4)},
		},
	}) {
		t.Fatalf("unexpected row(0): %s", spew.Sdump(row))
	}

	// Verify the cpu region=north is emitted next.
	if row := e.Emit(); !deep.Equal(row, &models.Row{
		Name:    "cpu",
		Tags:    map[string]string{"region": "north"},
		Columns: []string{"col1", "col2"},
		Values: [][]interface{}{
			{time.Unix(0, 0).UTC(), nil, float64(4)},
		},
	}) {
		t.Fatalf("unexpected row(1): %s", spew.Sdump(row))
	}

	// Verify the mem series is emitted last.
	if row := e.Emit(); !deep.Equal(row, &models.Row{
		Name:    "mem",
		Columns: []string{"col1", "col2"},
		Values: [][]interface{}{
			{time.Unix(0, 4).UTC(), nil, float64(5)},
		},
	}) {
		t.Fatalf("unexpected row(2): %s", spew.Sdump(row))
	}

	// Verify EOF.
	if row := e.Emit(); row != nil {
		t.Fatalf("unexpected eof: %s", spew.Sdump(row))
	}
}

// Ensure the emitter will limit the chunked output from a series.
func TestEmitter_ChunkSize(t *testing.T) {
	// Build an emitter that pulls from one iterator with multiple points in the same series.
	e := influxql.NewEmitter([]influxql.Iterator{
		&FloatIterator{Points: []influxql.FloatPoint{
			{Name: "cpu", Tags: ParseTags("region=west"), Time: 0, Value: 1},
			{Name: "cpu", Tags: ParseTags("region=west"), Time: 1, Value: 2},
		}},
	}, true, 1)
	e.Columns = []string{"col1"}

	// Verify the cpu region=west is emitted first.
	if row := e.Emit(); !deep.Equal(row, &models.Row{
		Name:    "cpu",
		Tags:    map[string]string{"region": "west"},
		Columns: []string{"col1"},
		Values: [][]interface{}{
			{time.Unix(0, 0).UTC(), float64(1)},
		},
	}) {
		t.Fatalf("unexpected row(0): %s", spew.Sdump(row))
	}

	// Verify the cpu region=north is emitted next.
	if row := e.Emit(); !deep.Equal(row, &models.Row{
		Name:    "cpu",
		Tags:    map[string]string{"region": "west"},
		Columns: []string{"col1"},
		Values: [][]interface{}{
			{time.Unix(0, 1).UTC(), float64(2)},
		},
	}) {
		t.Fatalf("unexpected row(1): %s", spew.Sdump(row))
	}

	// Verify EOF.
	if row := e.Emit(); row != nil {
		t.Fatalf("unexpected eof: %s", spew.Sdump(row))
	}
}
