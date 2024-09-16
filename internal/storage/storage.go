package storage

var urlStore = map[string]string{}

func Save(key string, value string) {
	urlStore[key] = value
}

func Get(key string) string {
	return urlStore[key]
}
