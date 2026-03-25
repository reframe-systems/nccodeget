package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// --- Hardcoded configuration ---
// tableNamespace points to the FeatureScript element defining ncCodeTable.
// The m{mid} component is a microversion ID — update it if the library changes.
const (
	tableNamespace  = "d4c0229df8696d6cbb05b091c::vdafa713c7386f3b5456d95e8::e17715c365e85a6b492b54bba::m1fcf5770b05ba62d045677d9"
	tableType       = "ncCodeTable"
	tableParameters = "addPartNumbers=true;addMarkingsFirst=true"
)

// ---

func verify(test bool, format string, va ...any) {
	if !test {
		log.Fatalf(format, va...)
	}
}

func getAccessAndSecretKeys(filename string) (string, string) {
	bytes, err := os.ReadFile(filename)
	verify(err == nil, "failed to read %s", filename)
	var keys map[string]any
	err = json.Unmarshal(bytes, &keys)
	verify(err == nil, "error parsing JSON in %s", filename)
	secretKey, ok := keys["secretKey"].(string)
	verify(ok, "failed to retrieve secretKey")
	accessKey, ok := keys["accessKey"].(string)
	verify(ok, "failed to retrieve accessKey")
	return accessKey, secretKey
}

func parseOnshapePath(rawURL string) map[string]string {
	regex := regexp.MustCompile(`https://cad\.onshape\.com/documents/(\w+)/([wvm])/(\w+)/e/(\w+)`)
	m := regex.FindStringSubmatch(rawURL)
	verify(len(m) == 5, "failed to extract parameters from url: %s", rawURL)
	return map[string]string{
		"did":   m[1],
		"wvm":   m[2],
		"wvmid": m[3],
		"eid":   m[4],
	}
}

func apiGet(access, secret, endpoint string, params url.Values) []byte {
	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	verify(err == nil, "failed to build request: %v", err)
	req.Header.Set("Accept", "application/json;charset=UTF-8; qs=0.09")
	req.SetBasicAuth(access, secret)
	resp, err := http.DefaultClient.Do(req)
	verify(err == nil, "request failed: %v", err)
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	verify(err == nil, "failed to read response body: %v", err)
	return body
}

func getElementName(access, secret string, osPath map[string]string) string {
	endpoint := fmt.Sprintf(
		"https://cad.onshape.com/api/v14/documents/d/%s/%s/%s/elements",
		osPath["did"], osPath["wvm"], osPath["wvmid"],
	)
	body := apiGet(access, secret, endpoint, url.Values{})
	var elements []map[string]interface{}
	err := json.Unmarshal(body, &elements)
	verify(err == nil, "failed to parse elements response: %v", err)
	for _, el := range elements {
		if el["id"] == osPath["eid"] {
			name, ok := el["name"].(string)
			verify(ok, "element name not a string")
			return name
		}
	}
	log.Fatalf("element %s not found in document", osPath["eid"])
	return ""
}

func getFSTable(access, secret string, osPath map[string]string) []byte {
	endpoint := fmt.Sprintf(
		"https://cad.onshape.com/api/v14/partstudios/d/%s/%s/%s/e/%s/fstable",
		osPath["did"], osPath["wvm"], osPath["wvmid"], osPath["eid"],
	)
	params := url.Values{}
	params.Set("tableType", tableType)
	params.Set("tableNamespace", tableNamespace)
	params.Set("tableParameters", tableParameters)
	return apiGet(access, secret, endpoint, params)
}

// sanitizeTitle converts a table title like "14041, 14042" to "14041_14042".
func sanitizeTitle(title string) string {
	s := strings.ReplaceAll(title, ", ", "_")
	s = strings.ReplaceAll(s, ",", "_")
	s = strings.ReplaceAll(s, " ", "_")
	return s
}

func extractTableText(table map[string]interface{}) string {
	rows, _ := table["rows"].([]interface{})
	var lines []string
	for _, r := range rows {
		row, _ := r.(map[string]interface{})
		colToVal, _ := row["columnIdToValue"].(map[string]interface{})
		if val, ok := colToVal["ncCode"].(string); ok {
			lines = append(lines, val)
		}
	}
	return strings.Join(lines, "\n")
}

func writeFile(path, content string) {
	err := os.WriteFile(path, []byte(content+"\n"), 0644)
	verify(err == nil, "failed to write %s: %v", path, err)
	fmt.Println(" +", path)
}

func main() {
	dump := flag.Bool("dump", false, "write raw JSON response to <element-name>.json")
	flag.Parse()
	args := flag.Args()
	verify(len(args) >= 1, "Usage: nccodeget [--dump] <part-studio-url> [output-dir]")

	rawURL := args[0]
	outputBase := "."
	if len(args) >= 2 {
		outputBase = args[1]
	}

	wd, err := os.Getwd()
	verify(err == nil, "unable to get working directory: %v", err)
	access, secret := getAccessAndSecretKeys(filepath.Join(wd, "secrets.json"))

	osPath := parseOnshapePath(rawURL)

	fmt.Println("fetching element name...")
	elementName := getElementName(access, secret, osPath)

	fmt.Println("fetching NC code tables...")
	body := getFSTable(access, secret, osPath)

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	verify(err == nil, "failed to parse fstable response: %v", err)

	//before continuing on with processing, attempt to dump raw
	outDir := filepath.Join(outputBase, elementName)
	err = os.MkdirAll(outDir, 0755)
	verify(err == nil, "failed to create output directory: %v", err)
	if *dump {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		writeFile(filepath.Join(outDir, elementName+".json"), string(pretty))
	}

	tables, _ := data["tables"].([]interface{})
	verify(len(tables) > 0, "no tables in response")

	if *dump {
		pretty, _ := json.MarshalIndent(data, "", "  ")
		writeFile(filepath.Join(outDir, elementName+".json"), string(pretty))
	}

	var allTexts []string
	for _, t := range tables {
		table, _ := t.(map[string]interface{})
		title, _ := table["title"].(string)
		verify(title != "", "table has empty title")
		text := extractTableText(table)
		writeFile(filepath.Join(outDir, sanitizeTitle(title)+".txt"), text)
		allTexts = append(allTexts, text)
	}

	if *dump {
		writeFile(filepath.Join(outDir, elementName+".txt"), strings.Join(allTexts, "\n"))
	}
	fmt.Printf("done: %d table(s) written to %s\n", len(tables), outDir)
}
