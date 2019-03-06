package goose

import (
	"testing"
)

func TestNormalizeMySQLDSN(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		desc        string
		in          string
		out         string
		expectedErr string
	}{
		{
			desc:        "errors if dsn is invalid",
			in:          "root:password@tcp(mysql:3306)", // forgot the database name
			expectedErr: "invalid DSN: missing the slash separating the database name",
		},
		{
			desc: "works when there are no query parameters supplied with the dsn",
			in:   "root:password@tcp(mysql:3306)/db",
			out:  "root:password@tcp(mysql:3306)/db?parseTime=true",
		},
		{
			desc: "works when parseTime is already set to true supplied with the dsn",
			in:   "root:password@tcp(mysql:3306)/db?parseTime=true",
			out:  "root:password@tcp(mysql:3306)/db?parseTime=true",
		},
		{
			desc: "persists other parameters if they are present",
			in:   "root:password@tcp(mysql:3306)/db?allowCleartextPasswords=true&interpolateParams=true",
			out:  "root:password@tcp(mysql:3306)/db?allowCleartextPasswords=true&interpolateParams=true&parseTime=true",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			out, err := normalizeMySQLDSN(tc.in)
			if tc.expectedErr != "" {
				if err == nil {
					t.Errorf("expected an error, but did not have one, had (%#v, %#v)", out, err)
				} else if err.Error() != tc.expectedErr {
					t.Errorf("expected error %s but had %s", tc.expectedErr, err.Error())
				}
			} else if err != nil {
				t.Errorf("had unexpected error %s", err.Error())
			} else if out != tc.out {
				t.Errorf("had output mismatch, wanted %s but had %s", tc.out, out)
			}
		})
	}
}
