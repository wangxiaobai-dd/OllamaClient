package util

import (
	"log"
	"os"
	"regexp"
)

func ProcessResponse(content string) string {
	result := RemoveThinkTags(content)
	result = RemoveEmptyLine(result)
	return result
}

func RemoveThinkTags(content string) string {
	re := regexp.MustCompile(`(?s)<think>.*?</think>`)
	result := re.ReplaceAllString(content, "")
	return result
}

func RemoveEmptyLine(content string) string {
	reEmptyLine := regexp.MustCompile(`(?m)^\s*$\n?`)
	result := reEmptyLine.ReplaceAllString(content, "")
	return result
}

func WriteContentToFile(content, fileName string) {
	f, err := os.Create(fileName)
	if err != nil {
		log.Printf("failed to create file, err:%v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		log.Printf("failed to write to file, err:%v", err)
	}
}
