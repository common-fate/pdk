package publisher

type Publisher struct {
	ID      string   `json:"id" dynamodbav:"id"`
	Members []string `json:"members" dynamodbav:"members"`
}
