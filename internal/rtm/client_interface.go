package rtm

// RTMClientInterface defines the methods required for RTM API interactions.
// Both the real client and mock clients will implement this interface.
type RTMClientInterface interface {
	GetFrob() (string, error)
	GetToken(frob string) error
	GetAPIKey() string
	GetAuthToken() string
	SetAuthToken(token string)
	Sign(params map[string]string) string
	GetLists() ([]List, error)
}
