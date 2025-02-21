package util

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

func AddContentHeader(header, content string) string {
	return header + "\n" + content
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
	dir := filepath.Dir(fileName)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Printf("failed to makedir, err:%v", err)
		return
	}
	f, err := os.Create(fileName)
	if err != nil {
		log.Printf("failed to create file, err:%v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(content); err != nil {
		log.Printf("failed to write to file, err:%v", err)
	}
}

func UploadFile(serverURL, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	fileField, err := writer.CreateFormFile("file", filepath.Base(filePath))
	if err != nil {
		return fmt.Errorf("failed to create form file: %v", err)
	}

	_, err = io.Copy(fileField, file)
	if err != nil {
		return fmt.Errorf("failed to copy file content: %v", err)
	}
	writer.Close()

	response, err := http.Post(serverURL, writer.FormDataContentType(), &requestBody)
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned non-200 status: %d, response: %s", response.StatusCode, string(responseBody))
	}

	log.Printf("finish upload file, response:%s", string(responseBody))
	return nil
}

func ParseCronTime(cronTime string) (hour, minute, second int, err error) {
	parts := strings.Split(cronTime, ":")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("invalid time format, expected 'HH:MM:SS'")
	}
	hour, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid hour: %v", err)
	}
	minute, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minute: %v", err)
	}
	second, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid second: %v", err)
	}
	return hour, minute, second, nil
}
