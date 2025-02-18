package task

import (
	"OllamaChat/util"
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
	*option.CodeCheckOption
	diffMu    sync.Mutex
	diffFiles []string
	result    []string
}

func NewCodeCheckTask(option *option.CodeCheckOption) *CodeCheckTask {
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

func (ct *CodeCheckTask) getDiffFileName(revision string) string {
	fileName := fmt.Sprintf("%s/%s.diff", ct.DiffDir, revision)
	return fileName
}

func (ct *CodeCheckTask) getResultFileName(diffFile string) string {
	fileName := filepath.Base(diffFile)
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	outFile := baseName + "_analysis.text"
	return outFile
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
				diffFile := ct.getDiffFileName(revision)
				util.WriteContentToFile(out, diffFile)
				ct.diffMu.Lock()
				ct.diffFiles = append(ct.diffFiles, diffFile)
				ct.diffMu.Unlock()
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

func (ct *CodeCheckTask) Do(oc *ollama.OllamaClient) {
	ct.prepare()

	payload := oc.GetGeneratePayload()
	for _, file := range ct.diffFiles {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Printf("failed to read file, file:%s, err:%v", file, err)
			return
		}
		filePayload := *payload
		data := ollama.TemplateData{
			Content: string(content),
		}
		filePayload.Prompt, err = ollama.RenderPrompt(ct.Prompt, data)
		if err != nil {
			fmt.Printf("failed to render, file:%s, err:%v", file, err)
			return
		}
		response, err := oc.Generate(&filePayload)
		if err != nil {
			fmt.Printf("code check failed, file:%s,err:%v\n", file, err)
		} else {
			resultFile := ct.getResultFileName(file)
			util.WriteContentToFile(util.ProcessResponse(response), resultFile)
			fmt.Printf("code check success, file:%s\n", file)
		}
	}
}
