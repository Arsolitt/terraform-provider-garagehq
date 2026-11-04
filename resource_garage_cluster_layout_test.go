package main

import (
	"encoding/json"
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
