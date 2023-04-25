package publisher

type User struct {
	ID         string   `json:"id" dynamodbav:"id"`
	Email      string   `json:"email" dynamodbav:"email"`
	Publishers []string `json:"publishers" dynamodbav:"publishers"`
}
