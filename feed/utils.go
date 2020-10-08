package feed

const URL = "https://api.pro.coinbase.com/products/ETH-DAI/book?level=2"

func TransformToUpdate(input [][]interface{}) []*Update {
	newUpdates := make([]*Update, len(input))
	for idx, val := range input {
		newUpdates[idx] = &Update{
			Price: val[0].(string),
			Size:  val[1].(string),
		}
	}
	return newUpdates
}
