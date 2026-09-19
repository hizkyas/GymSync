// Package handlers — export_test.go
// Compiled only during `go test`. Exposes unexported symbols for white-box
// testing without breaking the production API surface.
package handlers

import "github.com/pashagolub/pgxmock/v3"

// EvaluateAccess exposes the package-private evaluateAccess function for
// unit testing all subscription-status branches without a real DB.
var EvaluateAccess = evaluateAccess

// NewCheckInHandlerWithPool creates a CheckInHandler backed by a pgxmock pool.
// pgxmock.PgxPoolIface satisfies DBQuerier because it implements QueryRow,
// Query, and Exec with the same signatures as *pgxpool.Pool.
func NewCheckInHandlerWithPool(mock pgxmock.PgxPoolIface) *CheckInHandler {
	return newCheckInHandlerFromQuerier(mock)
}
