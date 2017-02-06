/*
	TODO list of desired functionality:
	-Timers for when the instant world quest completion is up again.
	-Indicator for all characters who haven't completed weekly quests/world bosses.
	-Search for items and rep across all characters on a realm.
	-Price search through AH data (I think there's an API for this?)
	-Timers for when class hall researches and missions finish.
	-Automatically run AskMrRobot, then update Pawn with new weights.
	-Decide which world quests to run based on reward type, emmissaries, etc. (or just list interesting ones)
*/

package wow

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
)

type wowConfig struct {
	InstallLocation string      `json:"wowInstall"`
	BlizzAPIKey     string      `json:"apiKey"`
	Characters      []character `json:"characters"`
}

type character struct {
	Realm string `json:"realm"`
	Name  string `json:"name"`
}

var config = getConfig()

func init() {
	/* I might not need an init at all, but in here we can:
	   -set up periodic parsing for data store (or should it just happen when the user searches for something?)
	*
	temp, _ := json.Marshal(config)
	fmt.Printf("config=%+v", string(temp))
	fmt.Println()
	*/
}

func Dispatch(args []string) []byte {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return addCharacter(args[1:])
		case "del":
			return deleteCharacter(args[1:])
		default:
			return errorJSON(errors.New("wowapi, requested api function not found"))
		}
	}
	return errorJSON(errors.New("wow api args were blank"))
}

func getConfig() wowConfig {
	raw, fileErr := ioutil.ReadFile("config/config.json")
	if fileErr != nil {
		fmt.Println("Error reading file. " + fileErr.Error())
	}

	var config wowConfig
	json.Unmarshal(raw, &config)
	return config
}

func addCharacter(charData []string) []byte {
	if len(charData) != 2 {
		return errorJSON(errors.New("wrong number of args, character not added"))
	}
	charData[0] = strings.Title(charData[0])
	charData[1] = strings.Title(charData[1])
	for _, char := range config.Characters {
		if charData[0] == char.Realm && charData[1] == char.Name {
			return errorJSON(errors.New("character already exists"))
		}
	}
	config.Characters = append(config.Characters, character{charData[0], charData[1]})
	if saveConfig() == "failure" {
		return errorJSON(errors.New("error saving config to file"))
	}
	response, err := json.Marshal(struct {
		Data string `json:"data"`
	}{"success, character added"})
	if err != nil {
		return errorJSON(err)
	}
	return response
}

func deleteCharacter(charData []string) []byte {
	if len(charData) != 2 {
		return errorJSON(errors.New("wrong number of args, character not deleted"))
	}
	charData[0] = strings.Title(charData[0])
	charData[1] = strings.Title(charData[1])

	for i, char := range config.Characters {
		if charData[0] == char.Realm && charData[1] == char.Name {
			config.Characters = append(config.Characters[:i], config.Characters[i+1:]...)
			if saveConfig() == "failure" {
				return errorJSON(errors.New("error saving config to file"))
			}
			response, err := json.Marshal(struct {
				Data string `json:"data"`
			}{"success, character deleted"})
			if err != nil {
				return errorJSON(err)
			}
			return response
		}
	}
	return errorJSON(errors.New("character not found"))
}

func saveConfig() string {
	data, marshalErr := json.MarshalIndent(config, "", "    ")
	if marshalErr != nil {
		return "failure"
	}
	writeErr := ioutil.WriteFile("config/config.json", data, 0644)
	if writeErr != nil {
		return "failure"
	}
	return "success"
}

func errorJSON(e error) []byte {
	response, err := json.Marshal(struct {
		Error string `json:"error"`
	}{e.Error()})
	if err != nil {
		panic("Error encoding the JSON error. Something went horribly wrong.")
	}
	return response
}
