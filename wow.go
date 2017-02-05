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

import "fmt"

type wowConfig struct {
	installLocation string
	blizzAPIKey     string
	characters      []character
}

type character struct {
	Realm string
	Name  string
}

var config = getConfig()

func init() {
	/* I might not need an init at all, but in here we can:
	   -set up periodic parsing for data store (or should it just happen when the user searches for something?)
	*/
	fmt.Println("wow init. " + config.installLocation + " " + config.blizzAPIKey + " " + config.characters[0].Realm)
}

func getConfig() wowConfig {
	return wowConfig{"/path/on/disk", "super-secret-api-key", []character{character{"realm", "char1"}, character{"realm", "char2"}}}
}

func PackageTest(names []string) string {
	answer := "zug zug"
	for _, name := range names {
		answer = answer + ", " + name
	}
	return answer
}
