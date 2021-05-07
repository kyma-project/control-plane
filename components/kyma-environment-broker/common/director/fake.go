package director

import "fmt"

type FakeClient struct {
	labels      map[string]string
	consoleURLs map[string]string
}

func NewFakeClient() *FakeClient {
	return &FakeClient{
		consoleURLs: make(map[string]string),
		labels:      make(map[string]string),
	}
}

func (f *FakeClient) GetConsoleURL(accountID, runtimeID string) (string, error) {
	return f.consoleURLs[runtimeID], nil
}

func (f *FakeClient) SetConsoleURL(runtimeID, url string) {
	f.consoleURLs[runtimeID] = url
}

func (f *FakeClient) SetLabel(accountID, runtimeID, key, value string) error {
	f.labels[f.labelKey(accountID, runtimeID, key)] = value
	return nil
}

func (f *FakeClient) GetLabel(accountID, runtimeID, key string) (string, bool) {
	k := f.labelKey(accountID, runtimeID, key)
	value, found := f.labels[k]
	return value, found
}

func (f *FakeClient) labelKey(accountID, runtimeID, key string) string {
	return fmt.Sprintf("%s/%s/%s", accountID, runtimeID, key)
}
