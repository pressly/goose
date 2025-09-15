package gooseutil

import (
	"database/sql"
	"fmt"
	"strings"
)

// FormatSQLResultInfo formats the result of a SQL operation into a string to use
// in logging: "rows affected: 1, last insert id: 2".
//
// It returns a string with the number of rows affected and the last insert id
// (each is skipped in result if not supported by DB). Returns an empty string
// if nothingIf nothing is supported.
func FormatSQLResultInfo(res sql.Result) string {
	resultDetails := []string{}
	if rowsAffected, err := res.RowsAffected(); err == nil {
		detail := fmt.Sprintf("rows affected: %d", rowsAffected)
		resultDetails = append(resultDetails, detail)
	}
	if lastInsertId, err := res.LastInsertId(); err == nil {
		detail := fmt.Sprintf("last insert id: %d", lastInsertId)
		resultDetails = append(resultDetails, detail)
	}
	return strings.Join(resultDetails, ", ")
}
