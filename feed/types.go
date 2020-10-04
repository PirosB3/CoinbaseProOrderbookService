package feed

type Update struct {
	Price string
	Size  string
}

type orderbookSortedKey struct {
	Value float64
	Key   string
}

type sortByOrderbookPrice []*orderbookSortedKey

func (a sortByOrderbookPrice) Len() int           { return len(a) }
func (a sortByOrderbookPrice) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a sortByOrderbookPrice) Less(i, j int) bool { return a[i].Value < a[j].Value }
