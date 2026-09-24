package repository

import (
	"database/sql"
	"errors"
	"fmt"
	"testing"
)

var errBoom = errors.New("boom")

func TestConvertRowAndMissingRow(t *testing.T) {
	conv := func(n int) *int { return &n }

	got, err := convertRow(7, nil, conv)
	if err != nil || got == nil || *got != 7 {
		t.Fatalf("convertRow success = (%v, %v)", got, err)
	}

	got, err = convertRow(1, errBoom, conv)
	if !errors.Is(err, errBoom) || got != nil {
		t.Fatalf("convertRow error = (%v, %v)", got, err)
	}

	// Create paths must not treat ErrNoRows as a missing record.
	got, err = convertRow(1, sql.ErrNoRows, conv)
	if !errors.Is(err, sql.ErrNoRows) || got != nil {
		t.Fatalf("convertRow ErrNoRows = (%v, %v)", got, err)
	}

	got, err = convertMissingRow(0, sql.ErrNoRows, conv)
	if err != nil || got != nil {
		t.Fatalf("convertMissingRow ErrNoRows = (%v, %v)", got, err)
	}

	wrapped := fmt.Errorf("query: %w", sql.ErrNoRows)
	got, err = convertMissingRow(0, wrapped, conv)
	if err != nil || got != nil {
		t.Fatalf("convertMissingRow wrapped ErrNoRows = (%v, %v)", got, err)
	}

	got, err = convertMissingRow(3, nil, conv)
	if err != nil || got == nil || *got != 3 {
		t.Fatalf("convertMissingRow success = (%v, %v)", got, err)
	}

	got, err = convertMissingRow(3, errBoom, conv)
	if !errors.Is(err, errBoom) || got != nil {
		t.Fatalf("convertMissingRow other error = (%v, %v)", got, err)
	}
}

func TestConvertRows(t *testing.T) {
	conv := func(n int) *int { return &n }

	got, err := convertRows([]int(nil), nil, conv)
	if err != nil || got == nil || len(got) != 0 {
		t.Fatalf("convertRows empty = (%#v, %v)", got, err)
	}

	got, err = convertRows([]int{1, 2}, nil, conv)
	if err != nil || len(got) != 2 || *got[0] != 1 || *got[1] != 2 {
		t.Fatalf("convertRows values = (%v, %v)", got, err)
	}

	got, err = convertRows([]int{1}, errBoom, conv)
	if !errors.Is(err, errBoom) || got != nil {
		t.Fatalf("convertRows error = (%v, %v)", got, err)
	}
}
