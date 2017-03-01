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
	"os"
	"strings"
	"sync"
	"time"

	wowlib "github.com/S-Porter/blizzard-api-client"
)

type wowConfig struct {
	InstallLocation string `json:"wowInstall"`
	BlizzAPIKey     string `json:"apiKey"`
	UpdateTimeout   int    `json:"updateTimeout"`
}
type characterData struct {
	Characters []character `json:"characters"`
}
type character struct {
	Realm        string               `json:"realm"`
	Name         string               `json:"name"`
	LastModified uint                 `json:"lastModified"`
	Items        []item               `json:"items"`
	Reputation   []*wowlib.Reputation `json:"reputation"`
	Notes        []string
}
type item struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
	//could get an id and pull the item icon from the API/wow folder?
}

var config = readConfig()
var charData = readCharData()
var client *wowlib.ApiClient
var mutex = &sync.Mutex{}

func init() {
	if config.BlizzAPIKey == "super-secret-api-key" {
		fmt.Println("you forgot to add a real API key to config.json")
	}
	var clientError error
	client, clientError = wowlib.NewApiClient("US", "")
	client.Secret = config.BlizzAPIKey
	if clientError != nil {
		fmt.Println(clientError.Error())
	}

	characters := charData.Characters
	for _, char := range characters {
		summary, err := client.GetCharacter(char.Realm, char.Name)
		if err != nil {
			fmt.Println(err.Error())
		}
		//the blizzard API returns lastModified in ms since epoch, div/1000 for seconds.
		if ((summary.LastModified - char.LastModified) / 1000) > 300 {
			modified := time.Unix(int64(char.LastModified)/1000, 0)
			fmt.Println(char.Name + " last modified " + modified.Format("Mon Jan 2, 3:04 PM") + ", updating...")
			go blizzUpdateRep(char.Realm, char.Name)
		}
	}
	/* Figure out the best way to keep data updated. I would like to avoid pulling data when the user clicks
	something, since there would be a ~1-2 second delay (at best). I could set up polling to check LastModified
	every few minutes, and then only pull the full dataset on modification. */
}

// Dispatch takes args from the url and sends them off to the proper fns.
func Dispatch(args []string) []byte {
	if len(args) > 0 {
		switch args[0] {
		case "addchar":
			return addCharacter(args[1:])
		case "delchar":
			return deleteCharacter(args[1:])
		case "listchars":
			return listCharacters()
		case "getdatastore":
			return allCharData()
		case "getrep":
			return getRep(args[1:])
		case "addnote":
			return addNote(args[1:])
		default:
			return errorJSON(errors.New("wowapi, requested api function not found"))
		}
	}
	return errorJSON(errors.New("wowapi, args were blank"))
}

func allCharData() []byte {
	data, err := json.Marshal(charData)
	if err != nil {
		return errorJSON(errors.New("error getting character data"))
	}
	return data
}

func addNote(args []string) []byte {
	if len(args) != 3 {
		return errorJSON(errors.New("wrong number of args, couldn't add note"))
	}
	for _, char := range charData.Characters {
		if strings.Title(args[0]) == char.Realm && strings.Title(args[1]) == char.Name {
			char.Notes = append(char.Notes, args[2])
			bytes, _ := json.Marshal(struct {
				Data string `json:"data"`
			}{"success"})
			return bytes
		}
	}
	return errorJSON(errors.New("requested character not in config"))
}

func getRep(args []string) []byte {
	if len(args) != 2 {
		return errorJSON(errors.New("wrong number of args, couldn't pull rep"))
	}
	//for testing purposes I want to limit the results to Legion rep only.
	legionReps := []int{1828, 1948, 1883, 1900, 1090, 1859, 1894}

	for _, char := range charData.Characters {
		if strings.Title(args[0]) == char.Realm && strings.Title(args[1]) == char.Name {
			relevantReps := []*wowlib.Reputation{}
			for _, rep := range char.Reputation {
				//ignore reps at standing=3, value=0 (absolute neutral) to save space
				if rep.Standing != 3 && rep.Value != 0 && intInSlice(rep.Id, legionReps) {
					relevantReps = append(relevantReps, rep)
				}
			}
			bytes, _ := json.Marshal(struct {
				Data []*wowlib.Reputation `json:"data"`
			}{relevantReps})
			return bytes
		}
	}
	return errorJSON(errors.New("requested character not in config"))
}

/* Update the given character reputations using the blizzard API */
func blizzUpdateRep(realm, name string) {
	for i, char := range charData.Characters {
		if realm == char.Realm && name == char.Name {
			response, err := client.GetCharacterWithFields(realm, name, []string{"reputation"})
			if err != nil {
				print(errorJSON(errors.New("error updating character information")))
				return
			}

			mutex.Lock()
			charData.Characters[i].Reputation = response.Reputation
			charData.Characters[i].LastModified = response.LastModified
			writeCharData()
			mutex.Unlock()
			fmt.Println("finished updating " + name)
		}
	}
}

//checks if there is any stored data on the given character.
func (data characterData) checkExists(realm, name string) bool {
	for _, savedChar := range data.Characters {
		if strings.Title(realm) == savedChar.Realm && strings.Title(name) == savedChar.Name {
			return true
		}
	}
	return false
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
	newChar := character{strings.Title(newData[0]), strings.Title(newData[1]), 0, []item{}, []*wowlib.Reputation{}, []string{}}
	if charData.checkExists(newChar.Realm, newChar.Name) {
		return errorJSON(errors.New("character already exists"))
	}
	charData.Characters = append(charData.Characters, newChar)
	if writeCharData() == "failure" {
		return errorJSON(errors.New("error saving character changes to file"))
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
	delChar := character{strings.Title(delData[0]), strings.Title(delData[1]), 0, []item{}, []*wowlib.Reputation{}, []string{}}
	for i, existingChar := range charData.Characters {
		if delChar.Realm == existingChar.Realm && delChar.Name == existingChar.Name {
			charData.Characters = append(charData.Characters[:i], charData.Characters[i+1:]...)
			if writeCharData() == "failure" {
				return errorJSON(errors.New("error saving character changes to file"))
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

	// override the config file API key with the env variable if present
	envAPIKey, ok := os.LookupEnv("BLIZZ_API_KEY")
	if ok {
		config.BlizzAPIKey = envAPIKey
	}
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

func intInSlice(a int, list []int) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
