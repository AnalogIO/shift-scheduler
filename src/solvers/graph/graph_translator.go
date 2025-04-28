package graph

import (
	"hash/fnv"
	"encoding/json"
	// "fmt"
	"math/rand"
	"os"
	"regexp"
	"sort"
	// "strconv"
	"strings"

	"github.com/Slug-Boi/aion-cli/src/forms"
)

type debug_data struct {
	Group_number string `json:"group_number"`
	Timeslot     map[string]string `json:"timeslot"`
	Hash_string map[string]string `json:"hash_string"`
	Hash_value map[string]uint32 `json:"hash_value"`
	Cost 	  map[string]float64 `json:"cost"`
}

// Translates data from the forms package to the graph package
func Translate(data []forms.Form) ([]Edge, int, map[int]forms.Form, map[int]string, map[string]float64) {
	// Create a map to store the user node to user data
	nodeToUser := map[int]forms.Form{}
	db_list := []debug_data{}

	// Timeslot node to Unix Time map (First value is starting time and second value is ending time)
	nodeToTime := map[int]string{}
	timeToNode := map[string]int{}

	// groupTimeslotCost is a map of group + Timeslot to the cost of the edge. Used in advanced view
	groupTimeslotCost := map[string]float64{}

	// Users are always node 1 to len(data.PollResults)
	userNodeInc := 1
	// Timeslots are always node len(data.PollResults) + 1 to len(data.PollResults) + len(participants.Votes)
	intialTimeslotNodeInc := len(data) + 1
	// Start at the initial timeslot node for the incrementor
	timeslotNodeInc := intialTimeslotNodeInc

	// Sort users by HashString to ensure consistent ordering when generating the concatenated string
	sort.Slice(data, func(i, j int) bool {
		return data[i].GroupNumber < data[j].GroupNumber
	})

	// Create base string for creating heuristic
	sb := strings.Builder{}

	// Get the string of all group inputs
	allStrings := BaseHashString(data, sb)

	graph := []Edge{}

	// Translate participants to source linked nodes
	for i, participant := range data {
		// create debug data frame
		db := debug_data{
			Group_number: participant.GroupNumber,
			Timeslot:     map[string]string{},
			Hash_string:  map[string]string{},
			Hash_value:   map[string]uint32{},
			Cost:         map[string]float64{},
		}

		// Add edge from source to participant
		graph = append(graph, Edge{From: 0, To: userNodeInc, Capacity: 1, Cost: 0})

		// Translate timeslots to participant linked nodes:
		// Caps are all the individual costs for each timeslot
		// SumCap is the sum of all the costs for each timeslot
		caps := map[string]float64{}
		sumCap := 0.0

		// Calculation is done this way to make sure that Want are all weighted equally
		// Can do and Cannot are weighted different between groups with the idea being that groups
		// with many wishes will get lower values overall on their wishes meaning they are more likely to get
		// their wishes granted as they are more flexible. Can do will always be weighted lower than cannot
		// due to the sum division. These calculations give a more fair distribution of timeslots between groups
		// and will incentivize groups to be more flexible with their timeslots and answer truthfully
		for timeslot, vote := range participant.Votes {
			// Add edge from participant to timeslot
			if _, ok := nodeToTime[timeslotNodeInc]; !ok {
				nodeToTime[timeslotNodeInc] = timeslot
				timeToNode[timeslot] = timeslotNodeInc
				timeslotNodeInc++
			}
			caps, sumCap = CostSummer(timeslot, vote, caps, sumCap)
		}
		if sumCap == 0 {
			sumCap = 1
		}
		timeslotNodeInc = intialTimeslotNodeInc
		// Add edge from participant to timeslot
		//TODO: Check that this still works now that caps is map and not a float slice
		for timeslot := range participant.Votes {
			heuristic, hash := HashHeuristic(participant.GroupNumber, timeslot, allStrings)
			groupTimeslotCost[participant.GroupNumber+timeslot] = caps[timeslot]/sumCap + heuristic
			graph = append(graph, Edge{From: userNodeInc, To: timeToNode[timeslot], Capacity: 1, Cost: (caps[timeslot] / sumCap) + heuristic})
			timeslotNodeInc++

			//Add debug data
			db.Timeslot[timeslot] = timeslot
			db.Hash_string[timeslot] = participant.GroupNumber + timeslot + allStrings
			db.Cost[timeslot] = groupTimeslotCost[participant.GroupNumber+timeslot]
			db.Hash_value[timeslot] = hash
		}

		if i != len(data)-1 {
			timeslotNodeInc = intialTimeslotNodeInc
		}

		db_list = append(db_list, db)

		// Add user to map and increment user node
		nodeToUser[userNodeInc] = participant
		userNodeInc++
	}

	// Link timeslots to sink
	for i := intialTimeslotNodeInc; i < timeslotNodeInc; i++ {
		graph = append(graph, Edge{From: i, To: timeslotNodeInc, Capacity: 1, Cost: 0})
	}
	//TODO: add if statement for debug data save as json
	db_data, _ := json.Marshal(db_list)
	err := os.WriteFile("debug_data.json", db_data, 0644)
	if err != nil {
		panic("cannot write debug data to file")
	}

	return graph, timeslotNodeInc, nodeToUser, nodeToTime, groupTimeslotCost
}

// TODO: Figure out if this is doable with a rolling hash function
func HashHeuristic(groupName, timeslot, FullHash string) (float64,uint32) {
	// Combine the two hash strings from input
	combined_str := groupName + timeslot + FullHash

	// convert to byte array
	combined := []byte(combined_str)

	// Create hash value of 32 bits
	hasher := fnv.New32()
	hasher.Write(combined)
	hash := hasher.Sum32()


	// hash, err := strconv.ParseInt(formated_hash[:8], 16, 64)
	// if err != nil {
	// 	panic(fmt.Sprintf("cannot convert hash to int", err, hash_hex))
	// }

	// The hash is used to seed the random number generator resulting in the same number every time
	random := rand.New(rand.NewSource(int64(hash)))

	// bound the random number between 0 and 0.5
	// Be careful this value needs to be small enough that it doesn't mess with the timeslot alloted costs
	// But also large enough that it has an effect on the floating point number comparison in golang
	random_float := (random.Float64() * 0.000005) + 0

	// convert the hash to a binary string of 10 bits by shifting
	// Then we convert the binary string to a float64 for the heuristic
	// The binary number has 54 0s in front of it to make it a decimal number of minimal size
	// This is to make the heuristic as small as possible to avoid messing with the flow algorithm

	//NOTE: This code is extemely cursed and broken I will start with random library and maybe circle back
	// To this later if I have time
	// parsed, _ := strconv.ParseUint(binaryConvert(hash), 2, 64)
	// println(parsed)

	//float := math.Float64frombits(parsed)

	return random_float, hash
}

// Generates a base string for hashing the heuristic
// Valid hash input strings must be 32 characters or less and only contain alphanumeric characters
// If the string is not valid then a random string is generated and used
func BaseHashString(data []forms.Form, sb strings.Builder) string {
	// compile regex for valid characters
	regex := regexp.MustCompile(`[a-zA-Z0-9]`)
	for i := 0; i < len(data); i++ {
		//TODO: Probably redo this to be more modular
		if data[i].HashString != "" && len(data[i].HashString) < 33 {
			// Check if the hash string is valid
			if regex.MatchString(data[i].HashString) {
				sb.WriteString(data[i].HashString)
			} else {
				sb.WriteString(data[i].GroupNumber + data[i].Timestamp)
			}
		} else {
			sb.WriteString(data[i].GroupNumber + data[i].Timestamp)
		}
	}

	return sb.String()
}

func CostSummer(timeslot, vote string, caps map[string]float64, sumCap float64) (map[string]float64, float64) {
	// Translate wishes to float values
	if vote == "Want" {
		caps[timeslot] = 0.0
		// Implicit cost of 0 added to sum
	} else if vote == "Can do" {
		caps[timeslot] = 10.0
		sumCap += 10.0
	} else {
		caps[timeslot] = 100.0
		sumCap += 100.0
	}

	return caps, sumCap
}
