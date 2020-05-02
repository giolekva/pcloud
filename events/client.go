package events

type query struct {
	query     string
	operation string
	variables string
}

// Implements EventStore
type GraphQLClient struct {
	apiAddr string
}

func NewGraphQLClient(apiAddr string) EventStore {
	return &GraphQLClient{apiAddr}
}

func (c *GraphQLClient) GetEventsInState(state EventState) ([]Event, error) {
	return nil, nil
}
