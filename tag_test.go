package main

import (
	"testing"
)

func TestGenKey(t *testing.T) {
	t.Log(genKey("Username"))
	t.Log(genKey("Password"))
	t.Log(genKey("RealName"))
	t.Log(genKey("ID"))
	t.Log(genKey("UserID"))
	t.Log(genKey("UserId"))
	t.Log(genKey("ACID"))
}

func TestParseTag(t *testing.T) {
	tags, err := parseTag(`"json:"ohh"form"wow" sdfsdf"`)
	if err != nil {
		t.Fatal(err)
	} else {
		for _, tag := range tags {
			t.Log(tag.Key, tag.Val)
		}
	}
}

func TestTagStr(t *testing.T) {
	tags := make(Tags, 0)
	tags.Append("json", "ohh")
	tags.Append("form", "wow")
	t.Log(tags.TagStr())
}

func TestLookup(t *testing.T) {
	tags := make(Tags, 0)
	tags.Append("gorm", "id")
	val, ok := tags.Lookup("gorm")
	if ok {
		t.Log(val)
	}
	val, ok = tags.Lookup("form")
	t.Log(val, ok)
}
