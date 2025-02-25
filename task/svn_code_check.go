package task

import (
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

	"OllamaClient/ollama"
	"OllamaClient/option"
	"OllamaClient/util"
	cron "github.com/robfig/cron/v3"
)

type CodeCheckTask struct {
	*option.CodeCheckOption
	diffMu       sync.Mutex
	diffFiles    []string
	result       []string
	submitMsgMap map[string]string
}

func NewCodeCheckTask(option *option.CodeCheckOption) *CodeCheckTask {
	return &CodeCheckTask{CodeCheckOption: option}
}

func (ct *CodeCheckTask) initTask() {
	ct.diffMu.Lock()
	ct.diffFiles = []string{}
	ct.diffMu.Unlock()
	ct.result = []string{}
	ct.submitMsgMap = map[string]string{}
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

type SvnSubmitLog struct {
	Entries []LogEntry `xml:"logentry"`
}

type LogEntry struct {
	Revision string `xml:"revision,attr"`
	Msg      string `xml:"msg"`
}

func (ct *CodeCheckTask) parseSVNLog(xmlData string) ([]string, error) {
	var logData SvnSubmitLog
	err := xml.Unmarshal([]byte(xmlData), &logData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse log xml: %v", err)
	}
	var revisions []string
	for _, entry := range logData.Entries {
		revisions = append(revisions, entry.Revision)
		ct.submitMsgMap[entry.Revision] = entry.Msg
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

func (ct *CodeCheckTask) getMergeResultFile() string {
	date := time.Now().Format("2006-01-02")
	outfile := ct.OutDir + ct.OutPrefix + "." + date
	return outfile
}

func (ct *CodeCheckTask) getResultFileByDiffFile(diffFile string) string {
	outFile := ct.OutDir + ct.getRevisionByDiffFile(diffFile) + "_analysis.text"
	return outFile
}

func (ct *CodeCheckTask) getDiffFileByRevision(revision string) string {
	fileName := fmt.Sprintf("%s/%s.diff", ct.DiffDir, revision)
	return fileName
}

func (ct *CodeCheckTask) getRevisionByDiffFile(diffFile string) string {
	fileName := filepath.Base(diffFile)
	baseName := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	return baseName
}

func (ct *CodeCheckTask) getSubmitMsgByRevision(revision string) string {
	return ct.submitMsgMap[revision]
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
				diffFile := ct.getDiffFileByRevision(revision)
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

func (ct *CodeCheckTask) processResponse(diffFile, content string) string {
	result := util.RemoveThinkTags(content)
	result = util.RemoveEmptyLine(result)
	revision := ct.getRevisionByDiffFile(diffFile)
	header := "REVISION:" + revision + "\t\t" + ct.getSubmitMsgByRevision(revision)
	result = util.AddContentHeader(header, result)
	return result
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

func (ct *CodeCheckTask) finish() {
	for _, file := range ct.diffFiles {
		if err := os.Remove(file); err != nil {
			log.Printf("failed to delete file %s: %v", file, err)
		}
	}
}

func (ct *CodeCheckTask) exec(oc *ollama.OllamaClient) {
	resultFile := ct.getMergeResultFile()
	var result string
	payload := oc.GetGeneratePayload()
	for _, diff := range ct.diffFiles {
		content, err := os.ReadFile(diff)
		if err != nil {
			log.Printf("failed to read file, diff:%s, err:%v", diff, err)
			continue
		}
		filePayload := *payload
		data := ollama.TemplateData{
			Content: string(content),
		}
		filePayload.Prompt, err = ollama.RenderPrompt(ct.Prompt, data)
		if err != nil {
			log.Printf("failed to render, diff:%s, err:%v", diff, err)
			continue
		}
		response, err := oc.Generate(&filePayload)
		if err != nil {
			log.Printf("code check failed, diff:%s,err:%v\n", diff, err)
		} else {
			result = result + ct.processResponse(diff, response) + "\n\n"
			log.Printf("code check success, diff:%s\n", diff)
		}
	}
	util.WriteContentToFile(result, resultFile)
	if ct.UploadURL != "" {
		err := util.UploadFile(ct.UploadURL, resultFile)
		if err != nil {
			log.Printf("failed to upload file, err:%v", err)
		}
	}
}

func (ct *CodeCheckTask) Do(oc *ollama.OllamaClient) {
	if ct.CronTime != "" {
		c := cron.New(cron.WithSeconds())
		_, err := c.AddFunc(ct.CronTime, func() {
			log.Println("cron check code task exec")
			ct.prepare()
			ct.exec(oc)
			ct.finish()
		})
		if err != nil {
			log.Printf("failed to add cron job: %v", err)
		}
		c.Start()
	} else {
		ct.prepare()
		ct.exec(oc)
		ct.finish()
	}
}
