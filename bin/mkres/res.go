package main

// Make Android SQL resources for bus lines from row text.
// Raw text is a copy of the PDF content using Acrobat Reader
// (evince has some issues with copy/paste schedules).

import (
	"bufio"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"

	"github.com/matm/gogeo"
)

const (
	DEBUG  = false
	INDENT = 2
)

const (
	DBSTRUCT              = ""
	DFLT_CIRC_POLICY      = "1-6"
	TIME_PAT              = "^\\d{1,2}:\\d{2}$"
	CIRC_PAT              = "^\\*(.*)\\*$"
	STOP_CIRC_PAT         = "^(\\d{1,2}:\\d{2})\\*(.*)\\*$"
	dfltCirculationPolicy = DFLT_CIRC_POLICY
	XML_HEADER            = `<?xml version="1.0" encoding="utf-8"?>
<!-- GENERATED AUTOMATICALLY BY THE makeres.py SCRIPT. DO NOT MODIFY! -->
`
	TMP_DIR     = "/tmp"
	SELF_SUFFIX = "_Self"
	// Line compiler
	LCOMPILER   = "../bin/bsc/bsc"
	CONFIG_FILE = "../local.properties"
	// Absolute path to lines definition
	LINES_SRC_DIR = ""
)

const (
	CHUNK_DB_FILE = "htdb-chunks.xml"
	CHUNK_PREFIX  = "htdb_chunk"
	CHUNK_SIZE    = 64 * 1024
)

const (
	GPS_CACHE_FILE = "gps.csv"
	GPS_RSRC_FILE  = "gps.xml"
	RAW_DB_FILE    = "htdb.sql"
	CHKSUM_FILE    = ".checksum"
	CHKSUM_DB_FILE = "dbversion.xml"
	DB_STATS_FILE  = "dbstats.xml"
	// Networks definition file
	NETWORKS_FILE = "networks.json"
)

const (
//g_cities = []
//g_prefilter = None
)

// read_config looks for a local.properties config file with
// a lines.dir entry, which is the path to bus lines schedules
// to use.
func read_config() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	conf := path.Join(wd, CONFIG_FILE)

	f, err := os.Open(conf)
	if err != nil {
		return "", err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), "lines.dir=") {
			return strings.Split(scanner.Text(), "=")[1], nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", errors.New("lines.dir entry not found in config file")
}

// get_cities_in_cache returns a list of cities from cacheFile.
func get_cities_in_cache(cacheFile string) ([]string, error) {
	ccities := make([]string, 0)
	f, err := os.Open(cacheFile)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		ccities = append(ccities, strings.Split(scanner.Text(), ";")[0])
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return ccities, nil
}

// fetch_gps_coords returns the GPS coordinates of city. It uses Google
// Geocoding API.
// See http://code.google.com/intl/fr/apis/maps/documentation/geocoding/
func fetch_gps_coords(city string) (*gogeo.GpsPoint, error) {
	g := gogeo.NewGoogleGeoCoder()
	// FIXME: remove Hérault, France
	point, err := g.Geocode(&gogeo.Location{city + ", Hérault, France"})
	if err != nil {
		return nil, err

	}
	return point, nil
}

// TODO
func makeSQL() {
}

// TODO
func parse(infile string) {
}

// smart_capitalize applies a simple capitalization algorithm, turning
// city names like "saint jean-de-védas" to  "Saint-Jean-de-Védas".
func smart_capitalize(name string) string {
	apos := []string{"L'", "D'"}
	// Add uppercase caracters on words
	cname := strings.Title(strings.Trim(name, " "))
	// Check for all '-' in name to put some capitals where needed
	bname := []byte(cname)
	r := regexp.MustCompile("-")
	dashes := r.FindAllStringIndex(cname, -1)
	idx := 0

	lowerChar := func(r rune) byte {
		return byte(unicode.ToLower(r))
	}
	if len(dashes) > 1 {
		switch len(dashes) {
		case 2:
			if len(cname[dashes[0][1]:dashes[1][0]]) < 3 {
				idx = dashes[0][1]
				bname[idx] = lowerChar(rune(cname[idx]))
			}
		case 3: // ex: Saint-Jean-De-Védas
			idx = dashes[1][1]
			bname[idx] = lowerChar(rune(cname[idx]))
		case 4:
			for k := 1; k <= 2; k++ {
				idx = dashes[k][1]
				bname[idx] = lowerChar(rune(cname[idx]))
			}
		}
	}

	// Always use l' or d' in lowercase
	for _, k := range dashes {
		for _, p := range apos {
			if p == strings.ToUpper(cname[k[1]:k[1]+2]) {
				bname[k[1]] = lowerChar(rune(cname[k[1]]))
			}
		}
	}

	return string(bname)
}

// get_md5 returns the md5sum of a file.
func get_md5(name string) (string, error) {
	f, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	data, err := ioutil.ReadAll(f)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", md5.Sum(data)), nil
}

// Checksum of the database is performed using a global checksum of
// all the raw/*/*.txt file, the DBSTRUCT content and the filter.map
// entries.
// Furthermore, the computed checksum is used to check if database
// rebuilding is needed by the build system.
func compute_db_checksum(srcdir string) (string, error) {
	sources := make([]string, 0)
	fn := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(info.Name(), ".txt") {
			sources = append(sources, path)
		}
		return nil
	}
	if err := filepath.Walk(srcdir, fn); err != nil {
		return "", err
	}
	// Sort sources?

	return "", nil
}

// make_chunks splits the whole SQL dataset into smaller chunks of data.
// No split occurs if chunkSize is zero.
func make_chunks(rawName string, chunkSize int) (int, error) {
	createChunk := func(chunkCount int) (*os.File, error) {
		outname := path.Join(TMP_DIR, fmt.Sprintf("%s_%d.xml", CHUNK_PREFIX, chunkCount))
		fmt.Printf("[%-18s] new chunk file %s... ", fmt.Sprintf("chunk %02d", chunkCount), outname)
		out, err := os.Create(outname)
		if err != nil {
			return nil, err
		}
		_, err = out.Write([]byte(XML_HEADER))
		_, err = out.Write([]byte(`
<string name="ht_createdb">
`))
		if err != nil {
			return nil, err
		}
		return out, nil
	}

	chunk := 1
	out, err := createChunk(chunk)
	if err != nil {
		return 0, err
	}
	defer out.Close()

	seek := 0
	f, err := os.Open(rawName)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	// Order matters
	pats := [][2]string{
		{"--.*$", ""}, {"$", " "}, {"IS NULL;", "IS NULL## END;"},
		{"^[ \t]*", ""}, {"\n", ""}, {";", "\n"}, {"##", ";"},
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "BEGIN TRANSACTION;") ||
			strings.HasPrefix(line, "END TRANSACTION;") ||
			strings.HasPrefix(line, "END;") {
			continue
		}
		for _, pat := range pats {
			r, err := regexp.Compile(pat[0])
			if err != nil {
				return 0, err
			}
			line = r.ReplaceAllString(line, pat[1])
		}
		n, err := out.Write([]byte(line))
		if err != nil {
			return 0, err
		}
		seek += n
		if chunkSize > 0 && seek > chunkSize {
			seek = 0
			_, err := out.Write([]byte(`
</string>
`))
			if err != nil {
				return 0, err
			}
			fmt.Println("done.")

			// New chunk
			chunk++
			out, err = createChunk(chunk)
			if err != nil {
				return 0, err
			}
			defer out.Close()
		}

	}
	if err := scanner.Err(); err != nil {
		return 0, err
	}

	// Closing last chunk
	_, err = out.Write([]byte(`
</string>
`))
	if err != nil {
		return 0, err
	}
	fmt.Println("done.")

	return chunk, nil
}

// check_up_to_date checks if the source files are in sync with the current SQL and DB.
// Exits gracefully if nothing to do at all.
func check_up_to_date(chksum string) {
}

type BusNetwork struct {
}

// init_networks finds available bus networks. networks.json is a static file edited
// by hand. Returns a list of available networks ressources.
func init_networks(srcdir string) []BusNetwork {
	return nil
}

// bsc_compile runs the bsc compiler on *.in bus lines definitions. Returns the list of
// available bus networks.
func bsc_compile(srcdir string) []BusNetwork {
	return init_networks(srcdir)
}

// apply_prefilter applies any substitutions by using regexps defined in the prefilter
// file.
func apply_prefilter(prefilter, infile string) {
}

func main() {
	fmt.Println("WIP")
}
