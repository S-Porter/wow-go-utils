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
	InstallLocation string `json:"wowInstall"`
	BlizzAPIKey     string `json:"apiKey"`
}
type characterData struct {
	Characters []character `json:"characters"`
}
type character struct {
	Realm string `json:"realm"`
	Name  string `json:"name"`
	Items []item `json:"items"`
}
type item struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	//could get an id and pull the item icon from the API/wow folder?
}

var config = readConfig()
var charData = readCharData()

func init() {
	/* I might not need an init at all, but in here we can:
	   -set up periodic parsing for data store (or should it just happen when the user searches for something?)
	*
	temp, _ := json.Marshal(config)
	fmt.Printf("config=%+v", string(temp))
	fmt.Println()
	*/
	temp, _ := json.Marshal(charData)
	fmt.Printf("charData=%+v", string(temp))
	fmt.Println()

}

// Dispatch takes args from the url and sends them off to the proper fns.
func Dispatch(args []string) []byte {
	if len(args) > 0 {
		switch args[0] {
		case "add":
			return addCharacter(args[1:])
		case "del":
			return deleteCharacter(args[1:])
		case "listchars":
			return listCharacters()
		case "getDataStore":
			return []byte("hit data store endpoint...")
		default:
			return errorJSON(errors.New("wowapi, requested api function not found"))
		}
	}
	return errorJSON(errors.New("wowapi, args were blank"))
}

func listCharacters() []byte {
	response, err := json.Marshal(struct {
		Data []character `json:"data"` //TODO: change this to only marshal name/realm
	}{charData.Characters})
	if err != nil {
		return errorJSON(errors.New("error listing characters"))
	}
	return response
}

func addCharacter(newData []string) []byte {
	if len(newData) != 2 {
		return errorJSON(errors.New("wrong number of args, character not added"))
	}
	newChar := character{strings.Title(newData[0]), strings.Title(newData[1]), []item{}}
	for _, configChar := range charData.Characters {
		if newChar.Realm == configChar.Realm && newChar.Name == configChar.Name {
			return errorJSON(errors.New("character already exists"))
		}
	}
	charData.Characters = append(charData.Characters, newChar)
	if writeCharData() == "failure" {
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

func deleteCharacter(delData []string) []byte {
	if len(delData) != 2 {
		return errorJSON(errors.New("wrong number of args, character not deleted"))
	}
	delChar := character{strings.Title(delData[0]), strings.Title(delData[1]), []item{}}
	for i, existingChar := range charData.Characters {
		if delChar.Realm == existingChar.Realm && delChar.Name == existingChar.Name {
			charData.Characters = append(charData.Characters[:i], charData.Characters[i+1:]...)
			if writeCharData() == "failure" {
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

func readConfig() wowConfig {
	raw, fileErr := ioutil.ReadFile("config/config.json")
	if fileErr != nil {
		fmt.Println("Error reading config file. " + fileErr.Error())
	}
	var config wowConfig
	json.Unmarshal(raw, &config)
	return config
}

func writeConfig() string {
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

func readCharData() characterData {
	raw, fileErr := ioutil.ReadFile("data/characters.json")
	if fileErr != nil {
		fmt.Println("Error reading character data file. " + fileErr.Error())
	}
	var data characterData
	json.Unmarshal(raw, &data)
	return data
}

func writeCharData() string {
	data, marshalErr := json.MarshalIndent(charData, "", "    ")
	if marshalErr != nil {
		return "failure"
	}
	writeErr := ioutil.WriteFile("data/characters.json", data, 0644)
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
