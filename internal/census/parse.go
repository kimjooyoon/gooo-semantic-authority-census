package census

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

type semanticMap struct {
	values    map[string]string
	ambiguous map[string]bool
}

type irDocument struct {
	Activities []irActivity `json:"activities"`
}

type irActivity struct {
	ID       string `json:"id"`
	Semantic string `json:"semantic"`
}

func parseSource(path string) (semanticMap, error) {
	return parseLineBindings(path, "activity")
}

func parseGenerated(path string) (semanticMap, error) {
	return parseLineBindings(path, "// gooo-binding")
}

func parseLineBindings(path, prefix string) (semanticMap, error) {
	f, err := os.Open(path)
	if err != nil {
		return semanticMap{}, err
	}
	defer f.Close()
	out := semanticMap{values: map[string]string{}, ambiguous: map[string]bool{}}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || (strings.HasPrefix(line, "#") && prefix == "activity") {
			continue
		}
		fields := strings.Fields(line)
		prefixFields := strings.Fields(prefix)
		if len(fields) < len(prefixFields)+2 {
			continue
		}
		match := true
		for i := range prefixFields {
			if fields[i] != prefixFields[i] {
				match = false
			}
		}
		if !match {
			continue
		}
		id := fields[len(prefixFields)]
		semantic := strings.Join(fields[len(prefixFields)+1:], " ")
		if _, exists := out.values[id]; exists {
			out.ambiguous[id] = true
		}
		out.values[id] = semantic
	}
	return out, scanner.Err()
}

func parseIR(path string) (semanticMap, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return semanticMap{}, err
	}
	if err := validateUniqueJSONKeys(data); err != nil {
		return semanticMap{}, fmt.Errorf("decode semantic IR: %w", err)
	}
	var doc irDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return semanticMap{}, err
	}
	out := semanticMap{values: map[string]string{}, ambiguous: map[string]bool{}}
	for _, activity := range doc.Activities {
		if activity.ID == "" {
			return semanticMap{}, fmt.Errorf("IR activity id is empty")
		}
		if _, exists := out.values[activity.ID]; exists {
			out.ambiguous[activity.ID] = true
		}
		out.values[activity.ID] = activity.Semantic
	}
	return out, nil
}

func validateUniqueJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := walkJSONValue(decoder); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("trailing JSON value")
		}
		return err
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return fmt.Errorf("object key is not a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[key] = true
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case '[':
		for decoder.More() {
			if err := walkJSONValue(decoder); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		return fmt.Errorf("unexpected JSON delimiter %q", delim)
	}
}

func fileDigest(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}
