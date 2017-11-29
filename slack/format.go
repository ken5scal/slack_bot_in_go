package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

// JSONReader reads the value and converts it to JSON-encoded value
func JSONReader(v interface{}) (io.Reader, error) {
	buf := new(bytes.Buffer)
	err := json.NewEncoder(buf).Encode(v)

	if err != nil {
		return nil, fmt.Errorf("Failed to encode %v into JSON\n", v)
	}

	return buf, nil
}

// PrintIndentedJSON output indented json via stdout.
func PrintIndentedJSON(originalJSON interface{}) error {
	dataRaw, err := IndentedJSON(originalJSON)

	if err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, string(dataRaw))
	return nil
}

// IndentedJSON returns api.
func IndentedJSON(originalJSON interface{}) ([]byte, error) {
	return json.MarshalIndent(originalJSON, "", "    ")
}

// JSONBodyDecoder reads the next JSON-encoded value from its
// input and stores it in the value pointed to by out.
func JSONBodyDecoder(resp *http.Response, out interface{}) error {
	defer resp.Body.Close()
	return json.NewDecoder(resp.Body).Decode(out)
}
