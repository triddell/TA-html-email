package main

import (
	"compress/gzip"
	"encoding/csv"
	"io"
	"os"
	"strings"

	log "github.com/Sirupsen/logrus"
)

func extractCsvFromGz(resultsFilePath string, csvFilePath string) error {
	f, err := os.Open(resultsFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err.Error(),
			"results_file": resultsFilePath,
		}).Error("extractCsvFromGz: Open results file failed")
		return err
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		log.WithFields(log.Fields{
			"error":        err.Error(),
			"results_file": resultsFilePath,
		}).Error("extractCsvFromGz: Read results file failed")
		return err
	}

	outFile, err := os.Create(csvFilePath)
	if err != nil {
		log.WithFields(log.Fields{
			"csv_file": csvFilePath,
			"error":    err.Error(),
		}).Error("extractCsvFromGz: Create CSV file failed")
		return err
	}
	defer outFile.Close()

	if _, err := io.Copy(outFile, gzf); err != nil {
		log.WithFields(log.Fields{
			"csv_file": csvFilePath,
			"error":    err.Error(),
		}).Error("extractCsvFromGz: Copy to CSV file failed")
		return err
	}
	return nil
}

func readCsvRows(csvFilePath string) ([][]string, error) {
	f, err := os.Open(csvFilePath)

	if err != nil {
		log.WithFields(log.Fields{
			"csv_file": csvFilePath,
			"error":    err.Error(),
		}).Error("readCsvRows: Open CSV file failed")
		return nil, err
	}
	defer f.Close()

	r := csv.NewReader(f)
	rows, err := r.ReadAll()

	if err != nil {
		log.WithFields(log.Fields{
			"csv_file": csvFilePath,
			"error":    err.Error(),
		}).Error("readCsvRows: Read CSV file failed")
		return nil, err
	}
	return rows, err
}

func removeCsvInternalColumns(rows [][]string) [][]string {
	var internalColumns []int
	for i := range rows {
		if i == 0 {
			for h, header := range rows[i] {
				//rows[0] = append(rows[0], "Total")
				if strings.HasPrefix(header, "__") {
					internalColumns = append(internalColumns, h)
				}
			}
		}
		rows[i] = removeColumnsFromRow(rows[i], internalColumns)
	}
	return rows
}

func removeColumnsFromRow(row []string, indexesToRemove []int) []string {
	var newRow []string
	for i, v := range row {
		if !intInSlice(i, indexesToRemove) {
			newRow = append(newRow, v)
		}
	}
	return newRow
}

func intInSlice(val int, slice []int) (exists bool) {
	exists = false
	for _, v := range slice {
		if val == v {
			exists = true
			return
		}
	}
	return
}

func writeCsvRows(cleanCsvFilePath string, rows [][]string) error {

	f, err := os.OpenFile(cleanCsvFilePath, os.O_CREATE|os.O_WRONLY, 0660)

	if err != nil {
		log.WithFields(log.Fields{
			"csv_file": cleanCsvFilePath,
			"error":    err.Error(),
		}).Error("writeCsvRows: Open CSV file failed")
		return err
	}
	defer f.Close()

	w := csv.NewWriter(f)
	err = w.WriteAll(rows)

	return nil
}
