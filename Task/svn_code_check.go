package Task

import (
	"bufio"
	"bytes"
	"encoding/xml"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"OllamaChat/Ollama"
	"OllamaChat/Option"
)

type CodeCheckTask struct {
	*Option.CodeCheckOption
	diffMu    sync.Mutex
	diffFiles []string
	result    []string
}

func NewCodeCheckTask(option *Option.CodeCheckOption) *CodeCheckTask {
	return &CodeCheckTask{CodeCheckOption: option}
}

func (ct *CodeCheckTask) initTask() {
	ct.diffMu.Lock()
	ct.diffFiles = []string{}
	ct.diffMu.Unlock()
	ct.result = []string{}
}

type ArgsMode int

const (
	NoExtraArgs    = 0
	AddRepoURL     = 1
	AddProjectPath = 2
)

func (ct *CodeCheckTask) buildSvnArgs(mode ArgsMode, args ...string) []string {
	cmdArgs := append(args, "--username", ct.UserName, "--password", ct.Password)
	if mode == AddRepoURL {
		cmdArgs = append(cmdArgs, ct.RepoURL)
	} else if mode == AddProjectPath {
		cmdArgs = append(cmdArgs, ct.ProjectPath)
	}
	return cmdArgs
}

func (ct *CodeCheckTask) execCmd(mode ArgsMode, args ...string) (string, error) {
	cmdArgs := ct.buildSvnArgs(mode, args...)
	cmd := exec.Command("svn", cmdArgs...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	log.Printf("execSvnCmd: %v", cmd)
	err := cmd.Run()
	return out.String(), err
}

type Log struct {
	Entries []LogEntry `xml:"logentry"`
}

type LogEntry struct {
	Revision string `xml:"revision,attr"`
}

func (ct *CodeCheckTask) parseSVNLog(xmlData string) ([]string, error) {
	var logData Log
	err := xml.Unmarshal([]byte(xmlData), &logData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log xml: %v", err)
	}
	var revisions []string
	for _, entry := range logData.Entries {
		revisions = append(revisions, entry.Revision)
	}
	return revisions, nil
}

func (ct *CodeCheckTask) updateCode() (string, error) {
	return ct.execCmd(AddProjectPath, "update")
}

func (ct *CodeCheckTask) getRevisions() ([]string, string, error) {
	end := time.Now()
	endDate := end.Format("2006-01-02")

	start := end.AddDate(0, 0, ct.CheckDay)
	startDate := start.Format("2006-01-02")

	if ct.CheckDay > 0 {
		endDate, startDate = startDate, endDate
	}

	out, err := ct.execCmd(AddRepoURL, "log", "-r", "{"+startDate+"}:{"+endDate+"}", "--xml")
	if err != nil {
		return nil, out, err
	}
	revisions, err := ct.parseSVNLog(out)
	return revisions, out, err
}

func (ct *CodeCheckTask) getPrevRevision(revision string) string {
	curRev, _ := strconv.Atoi(revision)
	return strconv.Itoa(curRev - 1)
}

func (ct *CodeCheckTask) writeDiffToFile(diffContent, revision string) {
	fileName := fmt.Sprintf("%s/%s.diff", ct.DiffDir, revision)
	outFile, err := os.Create(fileName)
	defer outFile.Close()
	if err != nil {
		log.Printf("failed to create file, err:%v", err)
		return
	}
	writer := bufio.NewWriter(outFile)
	scanner := bufio.NewScanner(strings.NewReader(diffContent))
	for scanner.Scan() {
		_, err := writer.WriteString(scanner.Text() + "\n")
		if err != nil {
			log.Printf("failed to write to file:%s, err: %v", fileName, err)
			continue
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("error while reading diff, file:%s, err:%v", fileName, err)
	}
	err = writer.Flush()
	if err != nil {
		log.Printf("failed to flush writer, file:%s, err:%v", fileName, err)
	}
	ct.diffMu.Lock()
	ct.diffFiles = append(ct.diffFiles, fileName)
	ct.diffMu.Unlock()
}

func (ct *CodeCheckTask) generateDiff(revisions []string) {
	var wg sync.WaitGroup
	for _, revision := range revisions {
		prevRev := ct.getPrevRevision(revision)
		out, err := ct.execCmd(AddRepoURL, "diff", "-r", prevRev+":"+revision)
		if err != nil {
			log.Printf("failed to get diff, out:%v, err:%v", out, err)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				ct.writeDiffToFile(out, revision)
			}()
		}
	}
	wg.Wait()
	log.Println("generate diff file success")
}

func (ct *CodeCheckTask) prepare() {
	ct.initTask()
	out, err := ct.updateCode()
	if err != nil {
		log.Printf("failed to update code, out:%v, err:%v", out, err)
		return
	}

	revisions, out, err := ct.getRevisions()
	if err != nil {
		log.Printf("failed to get revisions, out:%v, err:%v", out, err)
		return
	}
	log.Println("revisions:", revisions)

	ct.generateDiff(revisions)
	log.Println("diff files", ct.diffFiles)
}

func (ct *CodeCheckTask) writeRespToFile(file, response string) {
	fileName := filepath.Base(file)
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outFile := baseName + "_analysis.txt"
	f, err := os.Create(outFile)
	if err != nil {
		log.Printf("failed to create file, err:%v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(response); err != nil {
		log.Printf("failed to write to file, err:%v", err)
	}
}

func (ct *CodeCheckTask) BuildRequestPayload(oc *Ollama.OllamaClient) *Ollama.RequestPayload {
	payload := oc.GetRequestPayload()
	if len(ct.Format) != 0 {
		payload.Format = &Ollama.FormatSpec{
			Type:       "object",
			Properties: map[string]Ollama.FormatField{},
			Required:   []string{"format"},
		}
		for key, value := range ct.Format {
			s, ok := value.(string)
			if !ok {
				log.Printf("failed to convert value to string, key:%s", key)
			}
			field := Ollama.FormatField{Type: s}
			payload.Format.Properties[key] = field
			payload.Format.Required = append(payload.Format.Required, key)
		}
	}
	return payload
}

func (ct *CodeCheckTask) Do(oc *Ollama.OllamaClient) {
	ct.prepare()

	payload := ct.BuildRequestPayload(oc)
	for _, file := range ct.diffFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("failed to read file, file:%s, err:%v", file, err)
			return
		}

		filePayload := *payload
		data := Ollama.TemplateData{
			Content: string(content),
		}

		filePayload.Prompt, err = Ollama.RenderPrompt(ct.Prompt, data)
		if err != nil {
			fmt.Printf("failed to render, file:%s, err:%v", file, err)
			return
		}

		respChan, errChan := oc.Generate(&filePayload)
		select {
		case resp := <-respChan:
			if resp.Done {
				ct.writeRespToFile(file, resp.Response)
				fmt.Printf("code check success, file:%s\n", file)
				break
			} else {
				fmt.Printf("code check failed, file:%s\n", file)
				break
			}
		case err := <-errChan:
			fmt.Printf("failed to request, file:%s,err:%s\n", file, err)
			break
		}
	}
}
