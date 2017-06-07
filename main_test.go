package main

import (
	"reflect"
	"strings"
	"testing"
)

func TestSplitRoleVarsArgs(t *testing.T) {
	tests := []struct {
		argv []string
		role string
		vars []string
		args []string
	}{
		{
			argv: []string{"A"},
			role: "A",
			vars: []string{},
			args: nil,
		},

		{
			argv: []string{"A", "var1", "var2", "var3"},
			role: "A",
			vars: []string{"var1", "var2", "var3"},
			args: nil,
		},

		{
			argv: []string{"A", "var1", "var2", "var3", "--"},
			role: "A",
			vars: []string{"var1", "var2", "var3"},
			args: []string{},
		},

		{
			argv: []string{"A", "var1", "var2", "var3", "--", "/run", "hello", "world"},
			role: "A",
			vars: []string{"var1", "var2", "var3"},
			args: []string{"/run", "hello", "world"},
		},

		{
			argv: []string{"A", "--", "/run", "hello", "world"},
			role: "A",
			vars: []string{},
			args: []string{"/run", "hello", "world"},
		},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.argv, " "), func(t *testing.T) {
			role, vars, args := splitRoleVarsArgs(test.argv)

			if role != test.role {
				t.Error("bad role:", role)
			}

			if !reflect.DeepEqual(vars, test.vars) {
				t.Error("bad vars:", vars)
			}

			if !reflect.DeepEqual(args, test.args) {
				t.Error("bad args:", args)
			}
		})
	}
}
