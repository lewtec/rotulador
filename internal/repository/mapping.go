package repository

import (
	"database/sql"
	"errors"
)

// convertRow maps a sqlc row when err is nil. Any other error is returned unchanged.
func convertRow[Row any, Dom any](row Row, err error, conv func(Row) *Dom) (*Dom, error) {
	if err != nil {
		return nil, err
	}
	return conv(row), nil
}

// convertMissingRow is convertRow, except sql.ErrNoRows is a missing record: (nil, nil).
func convertMissingRow[Row any, Dom any](row Row, err error, conv func(Row) *Dom) (*Dom, error) {
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	return convertRow(row, err, conv)
}

// convertRows maps each row. On success the result is non-nil, including when rows is empty.
func convertRows[Row any, Dom any](rows []Row, err error, conv func(Row) *Dom) ([]*Dom, error) {
	if err != nil {
		return nil, err
	}
	out := make([]*Dom, len(rows))
	for i, row := range rows {
		out[i] = conv(row)
	}
	return out, nil
}
