package main

type payload struct {
	App           string
	Owner         string
	Configuration struct {
		Alert                string
		AlertColor           string `json:"alert_color"`
		Csv                  string
		Description          string
		DescriptionInclude   string `json:"description_include"`
		LogLevel             string `json:"log_level"`
		LogoLink             string `json:"logo_link"`
		LogoURL              string `json:"logo_url"`
		Name                 string
		Results              string
		ResultsColor         string `json:"results_color"`
		ResultsMaxRawGzBytes string `json:"results_max_raw_gz_bytes"`
		Signature            string
		SMTPFrom             string `json:"smtp_from"`
		SMTPHost             string `json:"smtp_host"`
		SMTPPort             string `json:"smtp_port"`
		SMTPVerify           string `json:"smtp_verify"`
		SMTPStarttls         string `json:"smtp_starttls"`
		SMTPTls              string `json:"smtp_tls"`
		SMTPTo               string `json:"smtp_to"`
		SMTPUsername         string `json:"smtp_username"`
		Source               string
		Table                string
		TableMaxCols         string `json:"table_max_cols"`
		TableMaxRows         string `json:"table_max_rows"`
		Title                string
		URLHostPort          string `json:"url_host_port"`
		URLScheme            string `json:"url_scheme"`
	}
	Result struct {
		Body []string
		To   string
	}
	ResultsFile string `json:"results_file"`
	ResultsLink string `json:"results_link"`
	SearchName  string `json:"search_name"`
	SearchURI   string `json:"search_uri"`
	ServerHost  string `json:"server_host"`
	ServerURI   string `json:"server_uri"`
	SessionKey  string `json:"session_key"`
	Sid         string
}
