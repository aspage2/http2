package hpack

import (
	"fmt"
	"strings"
)

type TableEntry struct {
	Key   string
	Value string
}

func (te TableEntry) Size() int {
	return 32 + len(te.Key) + len(te.Value)
}

type HeaderLookupTable struct {
	entries    []TableEntry
	lo         int
	numEntries int
	size       int
	maxSize    int
}

func NewHeaderLookupTable() *HeaderLookupTable {
	return &HeaderLookupTable{
		entries: make([]TableEntry, 32),
		lo:      0,

		// The number of entries in the table
		numEntries: 0,

		// Size in octets of this table
		size:    0,
		maxSize: 16536,
	}
}

func (dt *HeaderLookupTable) NumEntries() int {
	return len(StaticTable) + dt.numEntries
}

func (dt *HeaderLookupTable) SetMaxSize(ms int) {
	if ms > dt.maxSize {
		dt.maxSize = ms
		return
	}
	for dt.size > dt.maxSize {
		dt.Evict()
	}
	dt.maxSize = ms
}

func (dt *HeaderLookupTable) ForEach(f func(TableEntry)) {
	for i := range dt.numEntries {
		f(dt.entries[dt.Nth(i)])
	}
}

func (dt *HeaderLookupTable) Resize() {
	newEntries := make([]TableEntry, 2*len(dt.entries))
	i := 0
	dt.ForEach(func(te TableEntry) {
		newEntries[i] = te
		i += 1
	})
	dt.lo = 0
	dt.entries = newEntries
}

func (dt *HeaderLookupTable) Evict() (string, string, bool) {
	if dt.numEntries == 0 {
		return "", "", false
	}
	ret := dt.entries[dt.lo]
	dt.size -= ret.Size()
	dt.numEntries -= 1
	dt.lo = dt.Nth(1)
	return ret.Key, ret.Value, true
}

func (dt *HeaderLookupTable) Insert(key, value string) bool {
	te := TableEntry{key, value}
	s := te.Size()
	if s > dt.maxSize {
		return false
	}
	for dt.size+s > dt.maxSize {
		dt.Evict()
	}
	if dt.numEntries >= len(dt.entries) {
		dt.ExpandDynamicTable()
	}
	dt.entries[dt.NextOpen()] = te
	dt.size += s
	dt.numEntries += 1
	return true
}

func (tbl *HeaderLookupTable) ExpandDynamicTable() {
	newEntries := make([]TableEntry, 2*len(tbl.entries))

	for i := range tbl.numEntries {
		newEntries[i] = tbl.entries[tbl.Nth(i)]
	}
	tbl.lo = 0
}

func (dt *HeaderLookupTable) Nth(ind int) int {
	if ind >= len(dt.entries) {
		panic("nah, mang")
	}

	return (dt.lo + ind) % len(dt.entries)
}

func (dt *HeaderLookupTable) NextOpen() int {
	return dt.Nth(dt.numEntries)
}

func (dt *HeaderLookupTable) Lookup(ind int) (string, string, bool) {
	ind -= 1
	if ind < len(StaticTable) {
		te := StaticTable[ind]
		return te.Key, te.Value, true
	}
	ind -= len(StaticTable)

	if ind < dt.numEntries {
		te := dt.entries[dt.Nth(ind)]
		return te.Key, te.Value, true
	}
	return "", "", false
}

func (dt *HeaderLookupTable) String() string {
	var sb strings.Builder

	fmt.Fprintln(&sb, "----- Dynamic Table -----")
	dt.ForEach(func(te TableEntry) {
		fmt.Fprintf(&sb, "[s = %4d] %s: %v\n", te.Size(), te.Key, te.Value)
	})
	fmt.Fprintf(&sb, "Table Size: %d\n", dt.size)
	return sb.String()
}

func (tbl *HeaderLookupTable) Find(k, v string) (idx int, justKey bool) {
	idx = -1
	for i := 1; i < tbl.NumEntries()+1; i++ {
		ek, ev, _ := tbl.Lookup(i)
		if ek == k && ev == v {
			idx = i
			justKey = false
			return
		} else if ek == k && idx == -1 {
			idx = i
			justKey = true
		}
	}
	return
}

var StaticTable = []TableEntry{
	{":authority", ""},
	{":method", "GET"},
	{":method", "POST"},
	{":path", "/"},
	{":path", "/index.html"},
	{":scheme", "http"},
	{":scheme", "https"},
	{":status", "200"},
	{":status", "204"},
	{":status", "206"},
	{":status", "304"},
	{":status", "400"},
	{":status", "404"},
	{":status", "500"},
	{"accept-charset", ""},
	{"accept-encoding", "gzip, deflate"},
	{"accept-language", ""},
	{"accept-ranges", ""},
	{"accept", ""},
	{"access-control-allow-origin", ""},
	{"age", ""},
	{"allow", ""},
	{"authorization", ""},
	{"cache-control", ""},
	{"content-disposition", ""},
	{"content-encoding", ""},
	{"content-language", ""},
	{"content-length", ""},
	{"content-location", ""},
	{"content-range", ""},
	{"content-type", ""},
	{"cookie", ""},
	{"date", ""},
	{"etag", ""},
	{"expect", ""},
	{"expires", ""},
	{"from", ""},
	{"host", ""},
	{"if-match", ""},
	{"if-modified-since", ""},
	{"if-none-match", ""},
	{"if-range", ""},
	{"if-unmodified-since", ""},
	{"last-modified", ""},
	{"link", ""},
	{"location", ""},
	{"max-forwards", ""},
	{"proxy-authenticate", ""},
	{"proxy-authorization", ""},
	{"range", ""},
	{"referer", ""},
	{"refresh", ""},
	{"retry-after", ""},
	{"server", ""},
	{"set-cookie", ""},
	{"strict-transport-security", ""},
	{"transfer-encoding", ""},
	{"user-agent", ""},
	{"vary", ""},
	{"via", ""},
	{"www-authenticate", ""},
}
