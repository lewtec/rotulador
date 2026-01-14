package annotation

import (
	"fmt"
	"io"
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Meta struct {
		Description string `yaml:"description"`
	} `yaml:"meta"`
	Tasks          []*ConfigTask          `yaml:"tasks"`
	Authentication map[string]*ConfigAuth `yaml:"auth"`
	I18N           []ConfigI18N           `yaml:"i18n"`
}

type ConfigI18N struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

type ConfigAuth struct {
	Password string `yaml:"password"`
}

type ConfigTask struct {
	ID        string                  `yaml:"id"`
	Name      string                  `yaml:"name"`
	ShortName string                  `yaml:"short_name"`
	Type      string                  `yaml:"type"`
	If        map[string]string       `yaml:"if"`
	Classes   map[string]*ConfigClass `yaml:"classes"`
}

type ConfigClass struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Examples    []string `yaml:"examples"`
}

func LoadConfig(filename string) (*Config, error) {
	var ret Config
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := f.Close(); err != nil {
			log.Printf("error closing config file: %v", err)
		}
	}()
	data, err := io.ReadAll(f)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(data, &ret)
	if err != nil {
		return nil, err
	}
	_taskDict := map[string]string{}
	for _, task := range ret.Tasks {
		taskName := task.ID
		_, ok := _taskDict[taskName]
		if ok {
			return nil, fmt.Errorf("task with %s is defined twice", taskName)
		}
		_taskDict[taskName] = ""
		if task.Type == "" {
			task.Type = "class"
		}
		if task.ShortName == "" {
			task.ShortName = task.Name
		}
		if task.Classes == nil {
			task.Classes = getClassesFromClassType(task.Type)
		}
		if task.Classes == nil {
			return nil, fmt.Errorf("task %s does not have any classes or a compatible type", taskName)
		}
	}
	if len(ret.Authentication) == 0 {
		return nil, fmt.Errorf("no users specified")
	}
	// Load i18n strings from YAML config into default locale
	if len(ret.I18N) > 0 {
		for _, term := range ret.I18N {
			if term.Name == "" {
				return nil, fmt.Errorf("one i18n item is invalid: does not provide the name attribute")
			}
			if term.Value == "" {
				return nil, fmt.Errorf("one i18n item is invalid: does not provide the value attribute")
			}
			// Add to bundle as English messages
			if err := AddMessage("en", term.Name, term.Value); err != nil {
				log.Printf("Warning: failed to add i18n message %s: %v", term.Name, err)
			}
		}
		log.Printf("Loaded %d i18n strings from YAML config", len(ret.I18N))
	}
	for user := range ret.Authentication {
		if ret.Authentication[user].Password == "" {
			return nil, fmt.Errorf("user %s has a null password", user)
		}
	}
	return &ret, nil
}

func getClassesFromClassType(classType string) map[string]*ConfigClass {
	switch classType {
	case "boolean":
		return map[string]*ConfigClass{
			"true": {
				Name: "Yes",
			},
			"false": {
				Name: "No",
			},
		}
	case "rotation":
		return map[string]*ConfigClass{
			"ok": {
				Name:        "OK",
				Description: "Not rotated",
			},
			"h_inv": {
				Name:        "Invert X",
				Description: "Invert in horizontal axis",
			},
			"v_inv": {
				Name:        "Invert Y",
				Description: "Invert in vertical axis",
			},
			"+90": {
				Name:        "+90deg",
				Description: "Rotate 90 degrees horary",
			},
			"-90": {
				Name:        "-90deg",
				Description: "Rotate 90 degrees antihorary",
			},
			"180": {
				Name:        "180deg",
				Description: "Rotate 180 degrees",
			},
		}
	default:
		return nil
	}
}
