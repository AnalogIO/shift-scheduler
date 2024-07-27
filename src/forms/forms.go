package forms

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/gocarina/gocsv"
)

type Form struct {
	Timestamp   string `csv:"Timestamp"`
	GroupNumber string `csv:"Group Number"`
	HashString  string `csv:"Pseudo Lottery String"`
	Votes       map[string]string
}

type Config struct {
	DefaultSolver string `json:"DefaultSolver"`
	FormID        string `json:"formID"`
}

// Generated by curl-to-Go: https://curlconverter.com/go/

//	curl -L https://docs.google.com/spreadsheets/d/YOUR_SHEET_ID/export?exportFormat=csv
//
// TODO: Add optional flag for api key currently uses config file
func GetForm(conf Config, local ...bool) []Form {
	var form []Form

	// if local flag is not used then download the form
	if len(local) == 0 {
		println("Downloading form")
		url := fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/export?exportFormat=csv", conf.FormID)
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Fatal(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		// turn resp header into a byte array
		byteValue, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Fatal(err)
		}

		os.Remove("./form.csv")

		err = os.WriteFile("./form.csv", byteValue, 0644)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Open csv file
	file, err := os.OpenFile("./form.csv", os.O_RDONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	err = gocsv.UnmarshalFile(file, &form)
	if err != nil {
		log.Fatal("error in reading and unmarshaling CSV data.\nEither form ID is wrong or form data is corrupted:\n", err)
	}
	file.Seek(0, 0)

	var timeslots []string
	// Manual conversion of variable timeslots to struct
	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		// Split the header line
		header := scanner.Text()
		splitHeaders := strings.Split(header, ",")
		timeslots = append(timeslots, splitHeaders[3:]...)
	}
	inc := 0
	for scanner.Scan() {
		// Split the line
		line := scanner.Text()
		splitLine := strings.Split(line, ",")
		// Create a map to store the votes
		votes := make(map[string]string)
		for i, v := range splitLine[3:] {
			votes[timeslots[i]] = v
		}
		form[inc].Votes = votes
		inc++
	}

	return form

}

// parse json with golang https://tutorialedge.net/golang/parsing-json-with-golang/
func GetConfigFile(testing ...string) (Config, error) {
	var jsonFile *os.File
	if len(testing) > 0 {
		// Open test config file location
		var err error
		jsonFile, err = os.Open(testing[0])
		if err != nil {
			return Config{}, fmt.Errorf("error opening test config file: %v", err)
		}
	} else {

		userConf, err := os.UserConfigDir()
		if err != nil {
			return Config{}, fmt.Errorf("error getting user config directory: %v", err)
		}

		// Open config file
		jsonFile, err = os.Open(userConf + "/aion-cli/config.json")
		if err != nil {
			return Config{}, fmt.Errorf("error opening config file: %v", err)
		}
	}

	// read our opened jsonFile as a byte array.
	byteValue, err := io.ReadAll(jsonFile)
	if err != nil {
		return Config{}, fmt.Errorf("error reading config file: %v", err)
	}

	// initialize config var
	var conf Config

	json.Unmarshal(byteValue, &conf)

	return conf, nil
}
