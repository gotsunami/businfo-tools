//
// This is the Bus Schedules Compiler (BSC)
// It is useful to preprocess line schedules and output
// formatted data suited for later processing by the
// tools/makeres.py script. It makes handling schedules
// much safer and faster.
//
package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const (
	SEP             = " "
	COMPILER_HEADER = "# Generated by the bsc compiler. DO NOT EDIT!"
)

var (
	DAYS = []string{"L", "Ma", "Me", "J", "V", "S", "Sa", "D", "Di", "F"}

	// Usual days separators
	DAYS_S = []string{"à", "/"}

	// Mapping of possible duplicate days
	DUPS = map[string]string{"S": "Sa", "Sa": "S", "D": "Di", "Di": "D"}

	// Mapping between weekdays and the *num* format policy required by the
	// Android app
	HTMAP = map[string]string{
		"L":  "1",
		"Ma": "2",
		"Me": "3",
		"J":  "4",
		"V":  "5",
		"S":  "6",
		"Sa": "6",
		"D":  "7",
		"Di": "7",
		"F":  "r", // rest days
		"à":  "-",
		"/":  ",",
		"DF": "7,r",
	}

	// Bus line features mapping
	FEATMAP = map[string]string{"0": "", "NSCO": "S", "SCO": "s"}
)

type citySpot struct {
	city     string   // city name
	stations []string // city bus line stations
}

var cities []citySpot
var cur_city = ""
var dflt_circulation_pat = ""

// Looks for and element in a slice
func found(val string, a []string) bool {
	for _, e := range a {
		if e == val {
			return true
		}
	}
	return false
}

// Remove empty string elements
func remove_empty_strings(data []string) []string {
	keep := []string{}
	for _, e := range data {
		if s := strings.Trim(e, " "); len(s) > 0 {
			keep = append(keep, s)
		}
	}
	return keep
}

// Compute the list of days
// An input like this one:
//
// L à V LMaMeJ LàS
//
// will convert to a three element slice:
//
// [1-5 1,2,3,4 1-6]
func get_days(days string) []string {
	sdays := strings.Split(days, SEP)
	sdays = remove_empty_strings(sdays)

	res := make([][]string, 0)
	k := 0

	for k < len(sdays) {
		if k < len(sdays)-1 {
			if found(sdays[k+1], DAYS_S) {
				res = append(res, sdays[k:k+3])
				k += 3
				continue
			} else {
				res = append(res, []string{sdays[k]})
			}
		} else {
			res = append(res, []string{sdays[k]})
		}
		k++
	}

	// [[L à V] [LMaMeJ] [LàS]]

	// Find separators in strings with no spaces
	for k, pat := range res {
		for _, sep := range DAYS_S {
			if len(pat) == 1 {
				if strings.Contains(pat[0], sep) {
					z := strings.Split(pat[0], sep)
					// new slice with distinct, separated elements
					z = []string{z[0], sep, z[1]}
					res[k] = z
				}
			}
		}
	}

	// [[L à V] [LMaMeJ] [L à S]]

	// Remaining strings without special chars
	for k, pat := range res {
		if len(pat) == 1 {
			// Like LMaMeJ, no spaces in it, one block
			tmp := []string{}
			for _, day := range DAYS {
				if strings.Contains(pat[0], day) {
					// Special case: check for any day synonyms
					if _, has := DUPS[day]; has {
						if !found(DUPS[day], tmp) {
							tmp = append(tmp, day)
							tmp = append(tmp, "/")
						}
					} else {
						tmp = append(tmp, day)
						tmp = append(tmp, "/")
					}
				}
			}
			res[k] = tmp[:len(tmp)-1] // remove trailing /
		}
		k++
	}

	// [[L à V] [L / Ma / Me / J] [L à S]]

	fres := []string{}
	for _, pat := range res {
		tmp := ""
		for _, j := range pat {
			tmp += HTMAP[j]
		}
		fres = append(fres, tmp)
	}

	return fres
}

// Transpose features
// A string like
//   NSCO   SCO  0 0 SCO
// would return a 5 elements slice
//   [S s  s]
func get_features(feat string) []string {
	in := strings.Split(feat, SEP)
	in = remove_empty_strings(in)
	res := []string{}
	r, _ := regexp.Compile("\\[.\\]")
	for _, k := range in {
		res = append(res, FEATMAP[r.ReplaceAllString(k, "")])
	}
	return res
}

func print_scheds(scheds []string) {
	fmt.Printf(strings.Replace(strings.Join(scheds, "\n"), " ", ";", -1))
}

/* FIXME
func is_all_uppercase(line string) bool {
    return line == strings.ToUpper(line)
}
*/

// Find city name and related stations by reading
// lines of text
func parse_cities_stations(data string) {
	line := strings.Split(data, SEP)
	// Will hold the full city name
	city_s := []string{}
	r1, _ := regexp.Compile("[-*'.’]")
	r2, _ := regexp.Compile("[*]")
	r3, _ := regexp.Compile("[A-Z]")

	for _, ent := range line {
		tmp := r1.ReplaceAllString(ent, "")
		// Skip numbers and non-letter chars (/)
		up := strings.ToUpper(tmp)
		if _, err := strconv.Atoi(tmp); err != nil && r3.MatchString(up) && tmp == up {
			city_s = append(city_s, ent)
			if len(line) == 1 {
				// Only city name on line
				cur_city = strings.Join(city_s, " ")
				cur_city = r2.ReplaceAllString(cur_city, "")
				cities = append(cities, citySpot{city: cur_city, stations: []string{}})
			}
		} else {
			if len(city_s) > 0 {
				// Just got a city name on the line
				cur_city = strings.Join(city_s, " ")
				cur_city = r2.ReplaceAllString(cur_city, "")
				cities = append(cities, citySpot{city: cur_city, stations: []string{}})
				break
			}
		}
	}

	if len(line) > 1 && (strings.Join(city_s, " ") == strings.Join(line, " ")) {
		// Special case, only city name on line (may be several words)
		cur_city = strings.Join(city_s, " ")
		cur_city = r2.ReplaceAllString(cur_city, "")
		cities = append(cities, citySpot{city: cur_city, stations: []string{}})
		return
	}

	if len(city_s) > 0 {
		if len(line) > 1 {
			st := &cities[len(cities)-1].stations
			*st = append(*st, strings.Join(line[len(city_s):], " "))
		}
	} else {
		// Just a station name on the line
		st := &cities[len(cities)-1].stations
		*st = append(*st, strings.Join(line, " "))
	}
}

// Handle one direction at a time
func handle_direction(data []string) {
	scheds := make([]string, 0)
	days := make([]string, 0)
	features := make([]string, 0)
	max_sched_width := 0
	city_block := false

	sc, _ := regexp.Compile(".*\\d{1,2}:\\d{2}.*")
	cl, _ := regexp.Compile("\\[.\\]")

	for _, line := range data {
		if len(line) == 0 {
			continue
		}
		// Schedules
		if sc.MatchString(line) {
			// Should be a schedule line
			city_block = false
			// Clean up the line
			bkts := cl.FindAllString(line, -1)
			for _, b := range bkts {
				line = strings.Replace(line, b, "", -1) // Replace all
			}
			line = strings.Replace(line, "*", "", -1) // Remove all *
			scheds = append(scheds, line)
			if line_size := len(strings.Split(line, SEP)); line_size > max_sched_width {
				max_sched_width = line_size
			}
		} else if strings.Index(line, "days=") == 0 {
			// Parsing days line
			city_block = false
			days = get_days(line[5:])
		} else if strings.Index(line, "p=") == 0 {
			// Parsing features
			city_block = false
			if len(line) != 2 {
				features = get_features(line[2:])
			}
		} else if strings.Index(line, "c=") == 0 {
			// Parsing cities and stations
			city_block = true
			parse_cities_stations(line[2:])
		} else if city_block {
			parse_cities_stations(line)
		}
	}

	// Use default circulation pattern if days= is empty
	if len(days) == 0 {
		max := 0
		for _, scline := range scheds {
			scline_s := strings.Split(scline, SEP)
			if len(scline_s) > max {
				max = len(scline_s)
			}
		}
		for k := 0; k < max; k++ {
			days = append(days, dflt_circulation_pat)
		}
	}

	for j, scline := range scheds {
		scline_s := strings.Split(scline, SEP)
		k := 0
		if len(days) < len(scline_s) {
			fmt.Println("Error: less days than schedules!")
			fmt.Println("days=", len(days), days)
			fmt.Println("scheds=", len(scline_s), scline_s)
			fail(fmt.Sprintf("schedule %d: %d < %d", j+1, len(days), len(scline_s)))
		}
		for _, sc := range scline_s {
			if sc != "-" {
				scline_s[k] = sc + "*" + days[k]
				if k < len(features) {
					scline_s[k] += features[k]
				}
				scline_s[k] += "*"
			}
			k++
			scheds[j] = strings.Join(scline_s, SEP)
		}
	}

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("Recovering panic...")
			fmt.Printf("Last schedule line is: %s\n", data[len(data)-1])
			fmt.Println("Cities:")
			for _, d := range cities {
				fmt.Printf("+ %s\n", d.city)
				for _, station := range d.stations {
					fmt.Printf("|--- %s\n", station)
				}
			}
			fmt.Printf("Current direction schedules (%d lines)\n", len(scheds))
			for k, s := range scheds {
				fmt.Printf("[L%02d] %s\n", k+1, s)
			}
			panic(r)
		}
	}()

	fmt.Printf("\ndirection=\n")
	st := 0
	for _, d := range cities {
		fmt.Printf("\ncity=%s\n", d.city)
		for _, station := range d.stations {
			fmt.Printf("%s;%s\n", station, strings.Replace(scheds[st], SEP, ";", -1))
			st++
		}
	}

	if max_sched_width > len(days) {
		fail(fmt.Sprintf("more schedules entries (%d) than days definition (%d)!", max_sched_width, len(days)))
	}
}

func debug(msg string) {
	fmt.Println("=> " + msg)
}

func fail(msg string) {
	fmt.Fprintf(os.Stderr, "\nError: %s\n", msg)
	os.Exit(1)
}

func init() {
	sort.Strings(DAYS_S)
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Missing bus line argument")
		fmt.Printf("Usage: %s line.in\n", os.Args[0])
		os.Exit(2)
	}

	f, err := os.Open(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	data := make([]string, 0)
	for scanner.Scan() {
		data = append(data, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	// Break content into 2 directions for separate
	// processing
	directions := make([][]string, 0)
	dir1 := false
	dir2 := false
	for _, line := range data {
		line = strings.Trim(line, " ")
		if len(line) == 0 || strings.Index(line, "#") == 0 {
			continue
		}
		if strings.Index(line, "direction=") == 0 {
			if dir1 {
				dir1 = false
				dir2 = true
			} else {
				dir1 = true
			}
			directions = append(directions, make([]string, 0))
		} else {
			if dir1 || dir2 {
				t := &directions[len(directions)-1]
				*t = append(*t, line)
			}
		}
	}

	if len(directions) != 2 {
		fail(fmt.Sprintf("Missing direction: only have %d (should have 2)!", len(directions)))
	}

	// Parsing header
	header := make([]string, 0)
	header = append(header, COMPILER_HEADER)
	header = append(header, fmt.Sprintf("# Compile: bsc %s\n", path.Base(os.Args[1])))

	kws := []string{"name", "circulation", "color", "from", "to"}
	for _, line := range data {
		line = strings.Trim(line, " ")
		if len(line) == 0 || strings.Index(line, "#") == 0 {
			continue
		}
		if strings.Index(line, "direction=") == 0 {
			break
		}
		for _, kw := range kws {
			if strings.Index(line, kw+"=") == 0 {
				header = append(header, line)
				if kw == "circulation" {
					dflt_circulation_pat = line[len(kw)+1:]
				}
				break
			}
		}
	}

	fmt.Println(strings.Join(header, "\n"))

	for k, d := range directions {
		// Data direction can be handled separately
		// cities is resetted before handling a new direction
		cities = make([]citySpot, 0)
		handle_direction(d)
		// Looks for city duplicates
		dup := map[string]int{}
		for _, cit := range cities {
			if _, found := dup[cit.city]; !found {
				dup[cit.city] = 1
			} else {
				dup[cit.city]++
				fail(fmt.Sprintf("found duplicates in direction %d for city %s in %s\n",
					k+1, cit.city, os.Args[1]))
			}
		}
	}
}
