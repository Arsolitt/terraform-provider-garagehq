package main

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func TestNodeRoleChangeMarshalJSON(t *testing.T) {
	capacity := int64(1 << 30)
	remove := true
	cases := []struct {
		name   string
		change nodeRoleChange
		want   string
	}{
		{"storage", nodeRoleChange{Id: "n", Zone: "dc1", Tags: []string{"a"}, Capacity: &capacity}, `{"id":"n","zone":"dc1","tags":["a"],"capacity":1073741824}`},
		{"no tags", nodeRoleChange{Id: "n", Zone: "dc1", Capacity: &capacity}, `{"id":"n","zone":"dc1","tags":[],"capacity":1073741824}`},
		{"empty tags", nodeRoleChange{Id: "n", Zone: "dc1", Tags: []string{}}, `{"id":"n","zone":"dc1","tags":[]}`},
		{"gateway", nodeRoleChange{Id: "n", Zone: "dc1"}, `{"id":"n","zone":"dc1","tags":[]}`},
		{"remove", nodeRoleChange{Id: "n", Zone: "dc1", Remove: &remove}, `{"id":"n","remove":true}`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.change)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %s, want %s", got, tc.want)
			}
		})
	}
}

func TestBuildNodeRoles(t *testing.T) {
	oneGiB := int64(1 << 30)
	tests := []struct {
		name    string
		roles   []interface{}
		want    []nodeRoleChange
		wantErr string
	}{
		{
			name: "storage and gateway",
			roles: []interface{}{
				map[string]interface{}{"id": "a", "zone": "dc1", "capacity": "1G", "tags": []interface{}{"x"}},
				map[string]interface{}{"id": "b", "zone": "dc2", "capacity": "", "tags": []interface{}{}},
			},
			want: []nodeRoleChange{
				{Id: "a", Zone: "dc1", Tags: []string{"x"}, Capacity: &oneGiB},
				{Id: "b", Zone: "dc2", Tags: []string{}},
			},
		},
		{
			name: "invalid capacity",
			roles: []interface{}{
				map[string]interface{}{"id": "a", "zone": "dc1", "capacity": "1X", "tags": []interface{}{}},
			},
			wantErr: `node "a": invalid capacity format: "1X"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildNodeRoles(tc.roles)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("got error %v, want it to contain %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("got error %v, want none", err)
			}
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestValidateRoleIDs(t *testing.T) {
	role := func(id interface{}) map[string]interface{} {
		return map[string]interface{}{"id": id, "zone": "dc1"}
	}
	tests := []struct {
		name    string
		roles   interface{}
		wantErr string
	}{
		{"unique IDs", []interface{}{role("a"), role("b")}, ""},
		{"empty IDs", []interface{}{role(""), role("")}, ""},
		{"unknown IDs", []interface{}{role(nil), map[string]interface{}{"zone": "dc1"}}, ""},
		{"duplicate IDs", []interface{}{role("a"), role("b"), role("a")}, `duplicate node ID "a"`},
		{"not a list", "a", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRoleIDs(tc.roles)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("got error %v, want none", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Fatalf("got error %v, want it to contain %q", err, tc.wantErr)
			}
		})
	}
}

func TestLayoutRoleEqual(t *testing.T) {
	oneGiB := int64(1 << 30)
	twoGiB := int64(2 << 30)
	remove := true
	tests := []struct {
		name   string
		cur    layoutNodeRole
		change nodeRoleChange
		want   bool
	}{
		{
			"same storage role",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB, Tags: []string{"a", "b"}},
			nodeRoleChange{Id: "n", Zone: "dc1", Capacity: &oneGiB, Tags: []string{"a", "b"}},
			true,
		},
		{
			"same gateway role",
			layoutNodeRole{Id: "n", Zone: "dc1"},
			nodeRoleChange{Id: "n", Zone: "dc1"},
			true,
		},
		{
			"same gateway role with empty tags",
			layoutNodeRole{Id: "n", Zone: "dc1", Tags: []string{}},
			nodeRoleChange{Id: "n", Zone: "dc1", Tags: []string{}},
			true,
		},
		{
			"different zone",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			nodeRoleChange{Id: "n", Zone: "dc2", Capacity: &oneGiB},
			false,
		},
		{
			"capacity removed",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			nodeRoleChange{Id: "n", Zone: "dc1"},
			false,
		},
		{
			"capacity added",
			layoutNodeRole{Id: "n", Zone: "dc1"},
			nodeRoleChange{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			false,
		},
		{
			"different capacity",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			nodeRoleChange{Id: "n", Zone: "dc1", Capacity: &twoGiB},
			false,
		},
		{
			"different number of tags",
			layoutNodeRole{Id: "n", Zone: "dc1", Tags: []string{"a"}},
			nodeRoleChange{Id: "n", Zone: "dc1", Tags: []string{"a", "b"}},
			false,
		},
		{
			"different tag order",
			layoutNodeRole{Id: "n", Zone: "dc1", Tags: []string{"a", "b"}},
			nodeRoleChange{Id: "n", Zone: "dc1", Tags: []string{"b", "a"}},
			false,
		},
		{
			"removal",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			nodeRoleChange{Id: "n", Remove: &remove},
			false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := layoutRoleEqual(tc.cur, tc.change); got != tc.want {
				t.Errorf("got %t, want %t", got, tc.want)
			}
		})
	}
}

func TestRemovedNodeIDs(t *testing.T) {
	tests := []struct {
		name     string
		oldRoles []nodeRoleChange
		newRoles []nodeRoleChange
		want     []string
	}{
		{"none", []nodeRoleChange{{Id: "a"}, {Id: "b"}}, []nodeRoleChange{{Id: "a"}, {Id: "b"}}, nil},
		{"one leaves", []nodeRoleChange{{Id: "a"}, {Id: "b"}}, []nodeRoleChange{{Id: "b"}}, []string{"a"}},
		{"one joins", []nodeRoleChange{{Id: "a"}}, []nodeRoleChange{{Id: "a"}, {Id: "b"}}, nil},
		{"all leave", []nodeRoleChange{{Id: "a"}, {Id: "b"}}, nil, []string{"a", "b"}},
		{"none before", nil, []nodeRoleChange{{Id: "a"}}, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := removedNodeIDs(tc.oldRoles, tc.newRoles)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLayoutRoleState(t *testing.T) {
	oneGiB := int64(1 << 30)
	oneGiBPlusOne := int64(1<<30 + 1)
	tests := []struct {
		name       string
		node       layoutNodeRole
		configured string
		want       map[string]interface{}
	}{
		{
			"gateway",
			layoutNodeRole{Id: "n", Zone: "dc1"},
			"",
			map[string]interface{}{"id": "n", "zone": "dc1"},
		},
		{
			"keeps the configured notation",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			"1024M",
			map[string]interface{}{"id": "n", "zone": "dc1", "capacity": "1024M"},
		},
		{
			"falls back to the layout value",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB},
			"2G",
			map[string]interface{}{"id": "n", "zone": "dc1", "capacity": "1G"},
		},
		{
			"falls back to exact bytes",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiBPlusOne},
			"1G",
			map[string]interface{}{"id": "n", "zone": "dc1", "capacity": "1073741825"},
		},
		{
			"tags",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB, Tags: []string{"a", "b"}},
			"1G",
			map[string]interface{}{"id": "n", "zone": "dc1", "capacity": "1G", "tags": []string{"a", "b"}},
		},
		{
			"empty tags",
			layoutNodeRole{Id: "n", Zone: "dc1", Capacity: &oneGiB, Tags: []string{}},
			"1G",
			map[string]interface{}{"id": "n", "zone": "dc1", "capacity": "1G"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := layoutRoleState(tc.node, tc.configured)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
