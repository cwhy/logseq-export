package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
)

type page struct {
	filename   string
	attributes map[string]string
	text       string
}

func findMatchingFiles(rootPath string, substring string) ([]string, error) {
	var result []string
	err := filepath.WalkDir(rootPath, func(path string, d fs.DirEntry, walkError error) error {
		if walkError != nil {
			return walkError
		}
		if d.IsDir() {
			return nil
		}
		file, err := os.OpenFile(path, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		defer file.Close()
		fileScanner := bufio.NewScanner(file)
		for fileScanner.Scan() {
			line := fileScanner.Text()
			if strings.Contains(line, substring) {
				result = append(result, path)
				return nil
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func main() {
	graphPath := flag.String("graphPath", "", "[MANDATORY] Path to the root of your logseq graph containing /pages and /journals directories.")
	blogFolder := flag.String("blogFolder", "", "[MANDATORY] Folder where this program creates a new subfolder with public logseq pages.")
	unquotedProperties := flag.String("unquotedProperties", "", "comma-separated list of logseq page properties that won't be quoted in the markdown frontmatter, e.g. 'date,public,slug")
	flag.Parse()
	if *graphPath == "" || *blogFolder == "" {
		log.Println("mandatory argument is missing")
		flag.Usage()
		os.Exit(1)
	}
	publicFiles, err := findMatchingFiles(*graphPath, "public::")
	if err != nil {
		log.Fatalf("Error during walking through a folder %v", err)
	}
	for _, publicFile := range publicFiles {
		log.Printf("copying %q", publicFile)
		srcContent, err := readFileToString(publicFile)
		if err != nil {
			log.Fatalf("Error when reading the %q file: %v", publicFile, err)
		}
		_, name := filepath.Split(publicFile)
		page := parsePage(name, srcContent)
		result := transformPage(page)
		dest := filepath.Join(*blogFolder, result.filename)
		folder, _ := filepath.Split(dest)
		err = os.MkdirAll(folder, os.ModePerm)
		if err != nil {
			log.Fatalf("Error when creating parent directory for %q: %v", dest, err)
		}
		err = writeStringToFile(dest, render(result, parseUnquotedProperties(*unquotedProperties)))
		if err != nil {
			log.Fatalf("Error when copying file %q: %v", dest, err)
		}
	}
}

func parseUnquotedProperties(param string) []string {
	if param == "" {
		return []string{}
	}
	return strings.Split(param, ",")
}

func render(p page, dontQuote []string) string {
	attributeBuilder := strings.Builder{}
	for name, value := range p.attributes {
		if slices.Contains(dontQuote, name) {
			attributeBuilder.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		} else {
			attributeBuilder.WriteString(fmt.Sprintf("%s: %q\n", name, value))
		}
	}
	return fmt.Sprintf("---\n%s---\n%s", attributeBuilder.String(), p.text)
}

func readFileToString(src string) (string, error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer srcFile.Close()
	bytes, err := os.ReadFile(src)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func writeStringToFile(dest string, content string) error {
	err := os.WriteFile(dest, []byte(content), os.ModePerm)
	if err != nil {
		return err
	}

	return nil
}
