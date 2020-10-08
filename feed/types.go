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
