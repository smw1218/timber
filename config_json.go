package timber

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"reflect"
)

// Granulars are overriding levels that can be either
// package paths or package path + function name
type JSONGranular struct {
	Level string `xml:"level"`
	Path  string `xml:"path"`
}

type JSONProperty struct {
	Name  string `xml:"name"`
	Value string `xml:"value"`
}

type JSONFilter struct {
	Enabled    bool
	Tag        string
	Type       string
	Level      string
	Format     JSONProperty
	Properties []JSONProperty
	Granulars  []JSONGranular
}

type JSONConfig struct {
	Filters []JSONFilter
}

// Loads the configuration from an JSON file (as you were probably expecting)
func (t *Timber) LoadJSONConfig(filename string) (error) {
	if len(filename) <= 0 {
		return fmt.Errorf("Empty filename")
	}

	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("TIMBER! Can't load json config file: %s %v", filename, err)
	}
	defer file.Close()

	config := JSONConfig{}
	err = json.NewDecoder(file).Decode(&config)
	if err != nil {
		return fmt.Errorf("TIMBER! Can't parse json config file: %s %v", filename, err)
	}

	for _, filter := range config.Filters {
		if !filter.Enabled {
			continue
		}
		level := getLevel(filter.Level)
		formatter := getJSONFormatter(filter)
		if err != nil {
			return err
		}
		granulars := make(map[string]Level)
		for _, granular := range filter.Granulars {
			granulars[granular.Path] = getLevel(granular.Level)
		}
		configLogger := ConfigLogger{Level: level, Formatter: formatter, Granulars: granulars}

		switch filter.Type {
		case "console":
			configLogger.LogWriter = new(ConsoleWriter)
		case "socket":
			configLogger.LogWriter, err = getJSONSocketWriter(filter)
			if err != nil {
				return err
			}
		case "file":
			configLogger.LogWriter, err = getJSONFileWriter(filter)
			if err != nil {
				return err
			}
		default:
			log.Printf("TIMBER! Warning unrecognized filter in config file: %v\n", filter.Tag)
			continue
		}

		t.AddLogger(configLogger)
	}
	return nil
}

func getJSONFormatter(filter JSONFilter) LogFormatter {
	format := ""
	property := JSONProperty{}

	// If format field is set then use it's value, otherwise
	// attempt to get the format field from the filters properties
	if !reflect.DeepEqual(filter.Format, property) {
		format = filter.Format.Value
	} else {
		for _, prop := range filter.Properties {
			if prop.Name == "format" {
				format = prop.Value
			}
		}
	}

	// If empty format set the default as just the message
	if format == "" {
		format = "%M"
	}
	return NewPatFormatter(format)
}

func getJSONSocketWriter(filter JSONFilter) (LogWriter, error) {
	var protocol, endpoint string

	for _, property := range filter.Properties {
		if property.Name == "protocol" {
			protocol = property.Value
		} else if property.Name == "endpoint" {
			endpoint = property.Value
		}
	}

	if protocol == "" || endpoint == "" {
		return nil, fmt.Errorf("TIMBER! Missing protocol or endpoint for socket log writer")
	}
	return NewSocketWriter(protocol, endpoint)
}

func getJSONFileWriter(filter JSONFilter) (LogWriter, error) {
	filename := ""

	for _, property := range filter.Properties {
		if property.Name == "filename" {
			filename = property.Value
		}
	}
	if filename == "" {
		return nil, fmt.Errorf("TIMBER! Missing filename for file log writer")
	}
	return NewFileWriter(filename)
}
