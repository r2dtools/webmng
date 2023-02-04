package options

const (
	HttpPort  = "http_port"
	HttpsPort = "https_port"
)

func GetDefaults() map[string]string {
	defaults := make(map[string]string)
	defaults[HttpPort] = "80"
	defaults[HttpsPort] = "443"

	return defaults
}
