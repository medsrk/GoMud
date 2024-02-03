package configs

import (
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"github.com/volte6/mud/util"
	"gopkg.in/yaml.v2"
)

const defaultConfigPath = "_datafiles/config.yaml"

type config struct {
	MaxCPUCores                  int      `yaml:"MaxCPUCores"`
	FolderItemData               string   `yaml:"FolderItemData"`
	FolderAttackMessageData      string   `yaml:"FolderAttackMessageData"`
	FolderUserData               string   `yaml:"FolderUserData"`
	FolderTemplates              string   `yaml:"FolderTemplates"`
	FileAnsiAliases              string   `yaml:"FileAnsiAliases"`
	FileKeywords                 string   `yaml:"FileKeywords"`
	CarefulSaveFiles             bool     `yaml:"CarefulSaveFiles"`
	PVPEnabled                   bool     `yaml:"PVPEnabled"`
	XPScale                      float64  `yaml:"XPScale"`
	TurnMilliseconds             int      `yaml:"TurnMilliseconds"`
	RoundSeconds                 int      `yaml:"RoundSeconds"`
	RoundsPerAutoSave            int      `yaml:"RoundsPerAutoSave"`
	MaxMobBoredom                int      `yaml:"MaxMobBoredom"`
	TelnetPort                   int      `yaml:"TelnetPort"`                   // Port used to accept telnet connections
	WebPort                      int      `yaml:"WebPort"`                      // Port used for web requests
	NextRoomId                   int      `yaml:"NextRoomId"`                   // The next room id to use when creating a new room
	LootGoblinRoundCount         int      `yaml:"LootGoblinRoundCount"`         // How often to spawn a loot goblin
	LootGoblinMinimumItems       int      `yaml:"LootGoblinMinimumItems"`       // How many items on the ground to attract the loot goblin
	LootGoblinMinimumGold        int      `yaml:"LootGoblinMinimumGold"`        // How much gold on the ground to attract the loot goblin
	LootGoblinIncludeRecentRooms bool     `yaml:"LootGoblinIncludeRecentRooms"` // should the goblin include rooms that have been visited recently?
	LogIntervalRoundCount        int      `yaml:"LogIntervalRoundCount"`        // How often to report the current round number.
	Locked                       []string `yaml:"Locked"`                       // List of locked config properties that cannot be changed without editing the file directly.
	Seed                         string   `yaml:"Seed"`                         // Seed that may be used for generating content
	OnLoginCommands              []string `yaml:"OnLoginCommands"`              // Commands to run when a user logs in
	Motd                         string   `yaml:"Motd"`                         // Message of the day to display when a user logs in
	BannedNames                  []string `yaml:"BannedNames"`                  // List of names that are not allowed to be used

	OnDeathEquipmentDropChance float64 `yaml:"OnDeathEquipmentDropChance"` // Chance a player will drop a given piece of equipment on death
	OnDeathAlwaysDropBackpack  bool    `yaml:"OnDeathAlwaysDropBackpack"`  // If true, players will always drop their backpack items on death
	OnDeathXPPenalty           string  `yaml:"OnDeathXPPenalty"`           // Possible values are: none, level, 10%, 25%, 50%, 75%, 90%, 100%

	// Protected values
	turnsPerRound   int     // calculated and cached when data is validated.
	turnsPerSave    int     // calculated and cached when data is validated.
	turnsPerSecond  int     // calculated and cached when data is validated.
	roundsPerMinute float64 // calculated and cached when data is validated.
}

var (
	configData           config = config{}
	configDataLock       sync.RWMutex
	ErrInvalidConfigName = errors.New("invalid config name")
	ErrLockedConfig      = errors.New("config name is locked")
)

// Expects a string as the value. Will do the conversion on its own.
func SetVal(propName string, propVal string, force ...bool) error {

	if strings.EqualFold(propName, `locked`) {
		return ErrLockedConfig
	}

	for _, lockedProp := range configData.Locked {
		if strings.EqualFold(lockedProp, propName) {
			if len(force) < 1 || !force[0] {
				return ErrLockedConfig
			}
		}
	}

	typeSearchStructVal := reflect.ValueOf(configData)
	// Get the value and type of the struct
	//val := reflect.ValueOf(configData)
	typ := typeSearchStructVal.Type()
	// Iterate over all fields of the struct to find the correct name
	for i := 0; i < typeSearchStructVal.NumField(); i++ {
		if strings.EqualFold(typ.Field(i).Name, propName) {
			propName = typ.Field(i).Name
			break
		}
	}

	structValue := reflect.ValueOf(&configData)
	structValue = structValue.Elem()
	fieldValue := structValue.FieldByName(propName)

	if !fieldValue.IsValid() {
		return fmt.Errorf("no such field: %s in obj", propName)
	}

	if !fieldValue.CanSet() {
		return fmt.Errorf("cannot set field %s", propName)
	}

	switch fieldValue.Kind() {
	case reflect.String:
		fieldValue.SetString(propVal)

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intValue, err := strconv.ParseInt(propVal, 10, 64)
		if err != nil {
			return fmt.Errorf("field is an integer, but provided value is not: %s", propVal)
		}
		fieldValue.SetInt(intValue)
	case reflect.Bool:
		boolValue, err := strconv.ParseBool(propVal)
		if err != nil {
			return fmt.Errorf("field is a boolean, but provided value is not: %s", propVal)
		}
		fieldValue.SetBool(boolValue)
	case reflect.Float32, reflect.Float64:
		floatValue, err := strconv.ParseFloat(propVal, 64)
		if err != nil {
			return fmt.Errorf("field is a float, but provided value is not: %s", propVal)
		}
		fieldValue.SetFloat(floatValue)

	case reflect.Slice:
		sliceVal := strings.Split(propVal, `;`)
		fieldValue.Set(reflect.ValueOf(sliceVal))

	// Add cases for other types as needed

	default:
		return fmt.Errorf("unsupported field type: %s", fieldValue.Kind())
	}

	configData.validate()

	// save the new config.
	writeBytes, err := yaml.Marshal(configData)
	if err != nil {
		return err
	}

	return util.Save(configPath(), writeBytes, configData.CarefulSaveFiles)

}

// Get all config data in a map with the field name as the key for easy iteration
func (c config) AllConfigData() map[string]any {

	lockedLoookup := map[string]struct{}{
		`locked`: {},
	}
	for _, lockedProp := range configData.Locked {
		lockedLoookup[strings.ToLower(lockedProp)] = struct{}{}
	}

	output := map[string]any{}

	// Get the value and type of the struct
	items := reflect.ValueOf(c)
	typ := items.Type()

	// Iterate over all fields of the struct
	for i := 0; i < items.NumField(); i++ {
		if !items.Field(i).CanInterface() {
			continue
		}

		name := typ.Field(i).Name
		if name == `Locked` {
			continue
		}

		mapName := name

		if _, ok := lockedLoookup[strings.ToLower(name)]; ok {
			mapName = fmt.Sprintf(`%s (locked)`, name)
		}

		itm := items.Field(i)
		if itm.Type().Kind() == reflect.Slice {

			v := reflect.Indirect(itm)
			for j := 0; j < v.Len(); j++ {

				cmd := itm.Index(j).Interface().(string)

				if len(cmd) > 27 {
					cmd = cmd[0:27]
				}

				output[fmt.Sprintf(`%s.%d`, name, j)] = cmd
			}

		} else {
			output[mapName] = itm.Interface()
		}

	}

	return output
}

// Ensures certain ranges and defaults are observed
func (c *config) validate() {

	if c.MaxCPUCores < 0 {
		c.MaxCPUCores = 0 // default
	}

	if c.FolderItemData == `` {
		c.FolderItemData = `_datafiles/items` // default
	}

	if c.FolderAttackMessageData == `` {
		c.FolderAttackMessageData = `_datafiles/combat-messages` // default
	}

	if c.FolderUserData == `` {
		c.FolderUserData = `_datafiles/users` // default
	}

	if c.FolderTemplates == `` {
		c.FolderTemplates = `_datafiles/templates` // default
	}

	if c.FileAnsiAliases == `` {
		c.FileAnsiAliases = `_datafiles/ansi-aliases.yaml` // default
	}

	if c.FileKeywords == `` {
		c.FileKeywords = `_datafiles/keywords.yaml` // default
	}

	// Nothing to do with CarefulSaveFiles
	// Nothing to do with PVPEnabled

	if c.XPScale <= 0 {
		c.XPScale = 1.0 // default
	}

	if c.TurnMilliseconds < 10 {
		c.TurnMilliseconds = 100 // default
	}

	if c.RoundSeconds < 1 {
		c.RoundSeconds = 4 // default
	}

	if c.OnDeathEquipmentDropChance < 0.0 || c.OnDeathEquipmentDropChance > 1.0 {
		c.OnDeathEquipmentDropChance = 0.0 // default
	}

	// Nothing to do with OnDeathAlwaysDropBackpack

	c.OnDeathXPPenalty = strings.ToLower(c.OnDeathXPPenalty)

	if c.OnDeathXPPenalty != `none` && c.OnDeathXPPenalty != `level` {
		// If not a valid percent, set to default
		if !strings.HasSuffix(c.OnDeathXPPenalty, `%`) {
			c.OnDeathXPPenalty = `none` // default
		} else {
			// If not a valid percent, set to default
			percent, err := strconv.ParseInt(c.OnDeathXPPenalty[0:len(c.OnDeathXPPenalty)-1], 10, 64)
			if err != nil || percent < 0 || percent > 100 {
				c.OnDeathXPPenalty = `none` // default
			}
		}
	}

	if c.RoundsPerAutoSave < 1 {
		c.RoundsPerAutoSave = 900 // default of 15 minutes worth of rounds
	}

	if c.MaxMobBoredom < 1 {
		c.MaxMobBoredom = 150 // default
	}

	if c.TelnetPort < 1 {
		c.TelnetPort = 33333 // default
	}

	if c.WebPort < 1 {
		c.WebPort = 80 // default
	}

	if c.Seed == `` {
		c.Seed = `Mud` // default
	}

	// Nothing to do with NextRoomId

	if c.LootGoblinRoundCount < 10 {
		c.LootGoblinRoundCount = 10 // default
	}

	if c.LootGoblinMinimumItems < 1 {
		c.LootGoblinMinimumItems = 2 // default
	}

	if c.LootGoblinMinimumGold < 1 {
		c.LootGoblinMinimumGold = 100 // default
	}

	// nothing to do with LootGoblinIncludeRecentRooms

	if c.LogIntervalRoundCount < 0 {
		c.LogIntervalRoundCount = 0
	}

	// Nothing to do with Locked

	// Pre-calculate and cache useful values
	c.turnsPerRound = (c.RoundSeconds * 1000) / c.TurnMilliseconds
	c.turnsPerSave = c.RoundsPerAutoSave * c.turnsPerRound
	c.turnsPerSecond = 1000 / c.TurnMilliseconds
	c.roundsPerMinute = 60 / float64(c.RoundSeconds)
}

func (c config) GetDeathXPPenalty() (setting string, pct float64) {

	setting = c.OnDeathXPPenalty
	pct = 0.0

	if c.OnDeathXPPenalty == `none` || c.OnDeathXPPenalty == `level` {
		return setting, pct
	}

	percent, err := strconv.ParseInt(c.OnDeathXPPenalty[0:len(c.OnDeathXPPenalty)-1], 10, 64)
	if err != nil || percent < 0 || percent > 100 {
		setting = `none`
		pct = 0.0
		return setting, pct
	}

	pct = float64(percent) / 100.0

	return setting, pct
}

func (c config) TurnsPerRound() int {
	return c.turnsPerRound
}

func (c config) TurnsPerAutoSave() int {
	return c.turnsPerSave
}

func (c config) TurnsPerSecond() int {
	return c.turnsPerSecond
}

func (c config) MinutesToRounds(minutes int) int {
	return int(math.Ceil(c.roundsPerMinute * float64(minutes)))
}

func (c config) SecondsToRounds(seconds int) int {
	return int(math.Ceil(float64(seconds) / float64(c.RoundSeconds)))
}

func (c config) MinutesToTurns(minutes int) int {
	return int(math.Ceil(float64(minutes*60*1000) / float64(c.TurnMilliseconds)))
}

func (c config) SecondsToTurns(seconds int) int {
	return int(math.Ceil(float64(seconds*1000) / float64(c.TurnMilliseconds)))
}

func (c config) IsBannedName(name string) bool {

	var startsWith bool
	var endsWith bool

	name = strings.ToLower(strings.TrimSpace(name))

	for _, bannedName := range c.BannedNames {

		bannedName = strings.ToLower(bannedName)

		if strings.HasPrefix(bannedName, `*`) {
			endsWith = true
			bannedName = bannedName[1:]
		}

		if strings.HasSuffix(bannedName, `*`) {
			startsWith = true
			bannedName = bannedName[0 : len(bannedName)-1]
		}

		if startsWith && endsWith { // if it is contained anywhere
			if strings.Contains(name, bannedName) {
				return true
			}
		} else if startsWith { // if it starts with
			if strings.HasPrefix(name, bannedName) {
				return true
			}
		} else if endsWith { // if it ends with
			if strings.HasSuffix(name, bannedName) {
				return true
			}
		}
	}

	return false
}

func GetConfig() config {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	return configData
}

func configPath() string {
	configPath := os.Getenv(`CONFIG_PATH`)
	if configPath == `` {
		configPath = defaultConfigPath
	}
	return configPath
}

func ReloadConfig() error {

	configPath := util.FilePath(configPath())

	bytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	tmpConfigData := config{}
	err = yaml.Unmarshal(bytes, &tmpConfigData)
	if err != nil {
		return err
	}

	tmpConfigData.validate()

	configDataLock.Lock()
	defer configDataLock.Unlock()
	// Assign it
	configData = tmpConfigData

	return nil
}

func init() {
	if err := ReloadConfig(); err != nil {
		panic(err)
	}
}
