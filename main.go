package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	gomail "github.com/go-mail/mail"
	"github.com/matcornic/hermes/v2"
	log "github.com/sirupsen/logrus"
)

func main() {

	logger, _ := os.OpenFile(os.Getenv("SPLUNK_HOME")+"/var/log/splunk/html-email.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	defer logger.Close()
	log.SetOutput(logger)
	formatter := &log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
		DisableColors:   true,
	}
	log.SetFormatter(formatter)

	var err error
	defer func() {
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Fatal("Fatal Error")
		}
	}()

	if len(os.Args) > 1 && os.Args[1] == "--execute" {

		// Splunk sends payload as one line of JSON over stdin
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		payloadInput := scanner.Text()

		//Global vars
		var body hermes.Body
		var csvFilePath string
		var cleanCsvFilePath string
		var outros []string
		var processResults bool

		if scanner.Err() != nil {
			log.WithFields(log.Fields{
				"error": scanner.Err().Error(),
			}).Error("JSON payload missing")
			return
		}

		p := &payload{}

		err = json.Unmarshal([]byte(payloadInput), p)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error unmarshalling JSON payload")
			return
		}

		log.SetLevel(getLogLevel(p.Configuration.LogLevel))

		log.WithFields(log.Fields{
			"json_payload": payloadInput,
		}).Debug("Configuration")

		h := hermes.Hermes{
			Product: hermes.Product{
				Name:      p.Configuration.Name,
				Link:      p.Configuration.LogoLink,
				Logo:      p.Configuration.LogoURL,
				Copyright: "✉️ Sent using HTML Email",
			},
		}

		// Max Processing Size of Raw Gzipped Results

		// Max Inline Table Rows
		maxRawGzBytes, err := strconv.Atoi(p.Configuration.ResultsMaxRawGzBytes)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error converting string Max Raw GZ Bytes to integer")
			return
		}
		fi, err := os.Stat(p.ResultsFile)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error geting file size of raw search results")
			return
		}
		if fi.Size() <= int64(maxRawGzBytes) {
			processResults = true
		} else {
			outros = append(outros, "The search results were not processed due to the maximum raw search size byte limit: "+p.Configuration.ResultsMaxRawGzBytes)
		}

		if (p.Configuration.Csv == "1" || p.Configuration.Table == "1") && processResults {
			csvFilePath = os.Getenv("SPLUNK_HOME") + "/var/run/splunk/csv/" + p.Sid + ".csv"
			cleanCsvFilePath = csvFilePath + "x"
			err = extractCsvFromGz(p.ResultsFile, csvFilePath)
			defer os.Remove(csvFilePath)
			if err != nil {
				return
			}
			rows, err := readCsvRows(csvFilePath)
			if err != nil {
				return
			}
			rows = removeCsvInternalColumns(rows)
			err = writeCsvRows(cleanCsvFilePath, rows)
			defer os.Remove(cleanCsvFilePath)
			if err != nil {
				return
			}
		}

		switch p.Configuration.Source {

		case "source_markdown":

			var description string

			if p.Configuration.DescriptionInclude == "1" {
				intros := strings.Split(p.Configuration.Description, "\n")

				for i, intro := range intros {
					intros[i] = "> " + intro
				}

				description = strings.Join(intros, "\n") + "\n\n"
			}

			markdown := description + strings.Join(p.Result.Body[:], "\n") + "\n***\n"

			body = hermes.Body{
				FreeMarkdown: hermes.Markdown(markdown),
				Title:        p.Configuration.Title,
				Signature:    p.Configuration.Signature,
			}

		case "source_results":

			var actions []hermes.Action
			var intros []string
			var data [][]hermes.Entry

			if p.Configuration.Alert == "1" {

				//Parse URL from ResultsLink
				r, err := url.Parse(p.ResultsLink)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("Error parsing results_link URL")
					return
				}

				h := r.Host
				s := r.Scheme

				if p.Configuration.URLHostPort != "" {
					h = p.Configuration.URLHostPort
				}

				if p.Configuration.URLScheme != "" {
					s = p.Configuration.URLScheme
				}

				u := url.URL{
					Host:   h,
					Path:   "/app/search/alert",
					Scheme: s,
				}
				q := u.Query()
				q.Set("s", p.SearchURI)
				u.RawQuery = q.Encode()

				actions = append(actions, hermes.Action{
					Instructions: "To view the alert on Splunk, please click here:",
					Button: hermes.Button{
						Color: p.Configuration.AlertColor,
						Text:  "View Alert",
						Link:  u.String(),
					}})
			}

			if p.Configuration.Results == "1" {

				u, err := url.Parse(p.ResultsLink)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("Error parsing results_link URL")
					return
				}

				if p.Configuration.URLHostPort != "" {
					u.Host = p.Configuration.URLHostPort
				}

				if p.Configuration.URLScheme != "" {
					u.Scheme = p.Configuration.URLScheme
				}

				actions = append(actions, hermes.Action{
					Instructions: "To view the search results on Splunk, please click here:",
					Button: hermes.Button{
						Color: p.Configuration.ResultsColor,
						Text:  "View Search Results",
						Link:  u.String(),
					}})
			}

			if p.Configuration.DescriptionInclude == "1" {
				intros = strings.Split(p.Configuration.Description, "\n")
			}

			if p.Configuration.Csv == "1" && processResults {
				outros = append(outros, "The search results are attached in a file named \"results.csv\".")
			}

			if p.Configuration.Table == "1" && processResults {
				rows, err := readCsvRows(cleanCsvFilePath)
				if err != nil {
					return
				}

				// Max Inline Table Rows
				maxRows, err := strconv.Atoi(p.Configuration.TableMaxRows)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("Error converting string Table Max Rows to integer")
					return
				}

				// Max Inline Table Columns
				maxCols, err := strconv.Atoi(p.Configuration.TableMaxCols)
				if err != nil {
					log.WithFields(log.Fields{
						"error": err.Error(),
					}).Error("Error converting string Table Max Cols to integer")
					return
				}

				var headers []string
				var truncateRows bool
				for i := range rows {
					if i < maxRows+1 {
						if i == 0 {
							for _, value := range rows[i] {
								headers = append(headers, value)
							}
							continue
						}
						var entries []hermes.Entry
						for x, value := range rows[i] {
							if x < maxCols {
								entry := hermes.Entry{
									Key:   headers[x],
									Value: value,
								}
								entries = append(entries, entry)
							} else {
								if !truncateRows {
									outros = append(outros, "The number of inline table columns have been truncated to their maximum: "+p.Configuration.TableMaxCols)
									truncateRows = true
								}
								break
							}
						}
						data = append(data, entries)
					} else {
						outros = append(outros, "The number of inline table rows have been truncated to their maximum: "+p.Configuration.TableMaxRows)
						break
					}
				}
			}

			body = hermes.Body{
				Actions:   actions,
				Intros:    intros,
				Outros:    outros,
				Title:     p.Configuration.Title,
				Signature: p.Configuration.Signature,
				Table: hermes.Table{
					Data: data,
				},
			}

		}

		email := hermes.Email{Body: body}
		emailBody, err := h.GenerateHTML(email)

		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error generating HTML")
			return
		}

		// Configure Message
		m := gomail.NewMessage()
		m.SetHeader("From", p.Configuration.SMTPFrom)
		toAddresses := strings.Split(p.Configuration.SMTPTo, ",")
		m.SetHeader("To", toAddresses...)
		m.SetHeader("Subject", p.Configuration.Title)
		m.SetBody("text/html", emailBody)

		if p.Configuration.Csv == "1" && processResults {
			m.Attach(cleanCsvFilePath, gomail.Rename("results.csv"))
		}

		//Send Message
		port, err := strconv.Atoi(p.Configuration.SMTPPort)
		if err != nil {
			log.WithFields(log.Fields{
				"error": err.Error(),
			}).Error("Error converting string SMTP port to integer")
			return
		}

		var password string

		if p.Configuration.SMTPUsername != "" {

			//Get SMTP Password
			password, err = retrieveSMTPPassword(p.ServerURI, p.SessionKey)
			if err != nil {
				log.WithFields(log.Fields{
					"error": err.Error(),
				}).Error("Error retrieving SMTP password")
				return
			}
		}

		if p.Configuration.SMTPTls == "1" || p.Configuration.SMTPStarttls == "1" {

			var d *gomail.Dialer

			d = gomail.NewDialer(p.Configuration.SMTPHost, port, p.Configuration.SMTPUsername, password)

			if p.Configuration.SMTPStarttls == "1" {
				d.StartTLSPolicy = gomail.MandatoryStartTLS
			}

			if p.Configuration.SMTPVerify != "1" {
				d.TLSConfig = &tls.Config{InsecureSkipVerify: true}
			}

			s, err := d.Dial()
			if err != nil {
				log.WithFields(log.Fields{
					"app":         p.App,
					"error":       err.Error(),
					"owner":       p.Owner,
					"server_host": p.ServerHost,
					"search_name": p.SearchName,
					"sid":         p.Sid,
					"subject":     p.Configuration.Title,
					"to":          p.Configuration.SMTPTo,
				}).Error("Error connecting to SMTP server (secure)")
				return
			}

			if err := gomail.Send(s, m); err != nil {
				log.WithFields(log.Fields{
					"app":         p.App,
					"error":       err.Error(),
					"owner":       p.Owner,
					"server_host": p.ServerHost,
					"search_name": p.SearchName,
					"sid":         p.Sid,
					"subject":     p.Configuration.Title,
					"to":          p.Configuration.SMTPTo,
				}).Error("Error sending message to SMTP server (secure)")
				return
			}

		} else {

			d := gomail.Dialer{Host: p.Configuration.SMTPHost, Port: port, Username: p.Configuration.SMTPUsername, Password: password}
			if err := d.DialAndSend(m); err != nil {
				log.WithFields(log.Fields{
					"app":         p.App,
					"error":       err.Error(),
					"owner":       p.Owner,
					"server_host": p.ServerHost,
					"search_name": p.SearchName,
					"sid":         p.Sid,
					"subject":     p.Configuration.Title,
					"to":          p.Configuration.SMTPTo,
				}).Error("Error connecting to SMTP server and sending message (non-secure)")
				return
			}
		}

		log.WithFields(log.Fields{
			"app":         p.App,
			"owner":       p.Owner,
			"server_host": p.ServerHost,
			"search_name": p.SearchName,
			"sid":         p.Sid,
			"subject":     p.Configuration.Title,
			"to":          p.Configuration.SMTPTo,
		}).Info("HTML Email Sent")

		/*
			// Delete Attachments
			if (p.Configuration.Csv == "1" || p.Configuration.Table == "1") && processResults {
				os.Remove(csvFilePath)

				if err != nil {
					log.WithFields(log.Fields{
						"csv_file": csvFilePath,
						"error":    err.Error(),
					}).Error("Error removing temporary CSV file")
				}

				os.Remove(cleanCsvFilePath)

				if err != nil {
					log.WithFields(log.Fields{
						"clean_csv_file": cleanCsvFilePath,
						"error":          err.Error(),
					}).Error("Error removing temporary cleaned CSV file")
				}
			}
		*/

	} else {
		log.Error("Error: no \"--execute\" flag as first program argument")
	}

}

func getLogLevel(logLevel string) log.Level {
	switch strings.ToLower(logLevel) {
	case "debug":
		return log.DebugLevel
	case "info":
		return log.InfoLevel
	case "warn":
		return log.WarnLevel
	case "error":
		return log.ErrorLevel
	case "fatal":
		return log.FatalLevel
	default:
		return log.InfoLevel
	}
}
