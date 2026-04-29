package app

type Stats struct {
	InputBytes         int `json:"inputBytes"`
	OutputBytes        int `json:"outputBytes"`
	InputWords         int `json:"inputWords"`
	OutputWords        int `json:"outputWords"`
	InputApproxTokens  int `json:"inputApproxTokens"`
	OutputApproxTokens int `json:"outputApproxTokens"`
}
