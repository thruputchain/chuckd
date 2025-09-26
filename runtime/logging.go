package runtime

var logLevel string

func SetLogLevel(s string) {
	logLevel = s
}

func GetLogLevel() string {
	return logLevel
}
