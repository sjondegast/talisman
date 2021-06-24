package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"talisman/detector/severity"
	"talisman/gitrepo"
	"talisman/prompt"
	"talisman/talismanrc"
	"talisman/utility"

	"github.com/sirupsen/logrus"

	"gopkg.in/yaml.v2"

	"github.com/olekukonko/tablewriter"
)

type Details struct {
	Category string            `json:"type"`
	Message  string            `json:"message"`
	Commits  []string          `json:"commits"`
	Severity severity.Severity `json:"severity,omitempty"`
}

type ResultsDetails struct {
	Filename    gitrepo.FilePath `json:"filename"`
	FailureList []Details        `json:"failure_list"`
	WarningList []Details        `json:"warning_list"`
	IgnoreList  []Details        `json:"ignore_list"`
}

type FailureTypes struct {
	Filecontent int `json:"filecontent"`
	Filesize    int `json:"filesize"`
	Filename    int `json:"filename"`
	Warnings    int `json:"warnings"`
	Ignores     int `json:"ignores"`
}

type ResultsSummary struct {
	Types FailureTypes `json:"types"`
}

//DetectionResults represents all interesting information collected during a detection run.
//It serves as a collecting parameter for the tests performed by the various Detectors in the DetectorChain
//Currently, it keeps track of failures and ignored files.
//The results are grouped by FilePath for easy reporting of all detected problems with individual files.
type DetectionResults struct {
	mode    talismanrc.Mode
	Summary ResultsSummary   `json:"summary"`
	Results []ResultsDetails `json:"results"`
}

func (r *ResultsDetails) getWarningDataByCategoryAndMessage(failureMessage string, category string) *Details {
	detail := getDetailsByCategoryAndMessage(r.WarningList, category, failureMessage)
	r.WarningList = append(r.WarningList, *detail)
	return detail
}

func (r *ResultsDetails) getFailureDataByCategoryAndMessage(failureMessage string, category string) *Details {
	detail := getDetailsByCategoryAndMessage(r.FailureList, category, failureMessage)
	if detail == nil {
		detail = &Details{category, failureMessage, make([]string, 0), severity.Low}
		r.FailureList = append(r.FailureList, *detail)
	}
	return detail
}

func (r *ResultsDetails) addIgnoreDataByCategory(category string) {
	isCategoryAlreadyPresent := false
	for _, detail := range r.IgnoreList {
		if strings.Compare(detail.Category, category) == 0 {
			isCategoryAlreadyPresent = true
		}
	}
	if !isCategoryAlreadyPresent {
		detail := Details{category, "", make([]string, 0), severity.Low}
		r.IgnoreList = append(r.IgnoreList, detail)
	}
}

func getDetailsByCategoryAndMessage(detailsList []Details, category string, failureMessage string) *Details {
	for _, detail := range detailsList {
		if strings.Compare(detail.Category, category) == 0 && strings.Compare(detail.Message, failureMessage) == 0 {
			return &detail
		}
	}

	return nil
}

func (r *DetectionResults) getResultDetailsForFilePath(fileName gitrepo.FilePath) *ResultsDetails {
	for _, resultDetail := range r.Results {
		if resultDetail.Filename == fileName {
			return &resultDetail
		}
	}
	return nil
}

//NewDetectionResults is a new DetectionResults struct. It represents the pre-run state of a Detection run.
func NewDetectionResults(mode talismanrc.Mode) *DetectionResults {
	return &DetectionResults{
		mode,
		ResultsSummary{
			FailureTypes{0, 0, 0, 0, 0},
		},
		make([]ResultsDetails, 0),
	}

}

//Fail is used to mark the supplied FilePath as failing a detection for a supplied reason.
//Detectors are encouraged to provide context sensitive messages so that fixing the errors is made simple for the end user
//Fail may be called multiple times for each FilePath and the calls accumulate the provided reasons
func (r *DetectionResults) Fail(filePath gitrepo.FilePath, category string, message string, commits []string, severity severity.Severity) {
	isFilePresentInResults := false
	for resultIndex := 0; resultIndex < len(r.Results); resultIndex++ {
		if r.Results[resultIndex].Filename == filePath {
			isFilePresentInResults = true
			isEntryPresentForGivenCategoryAndMessage := false
			for detailIndex := 0; detailIndex < len(r.Results[resultIndex].FailureList); detailIndex++ {
				if strings.Compare(r.Results[resultIndex].FailureList[detailIndex].Category, category) == 0 && strings.Compare(r.Results[resultIndex].FailureList[detailIndex].Message, message) == 0 {
					isEntryPresentForGivenCategoryAndMessage = true
					r.Results[resultIndex].FailureList[detailIndex].Commits = append(r.Results[resultIndex].FailureList[detailIndex].Commits, commits...)
				}
			}
			if !isEntryPresentForGivenCategoryAndMessage {
				r.Results[resultIndex].FailureList = append(r.Results[resultIndex].FailureList, Details{category, message, commits, severity})
			}
		}
	}
	if !isFilePresentInResults {
		failureDetails := Details{category, message, commits, severity}
		resultDetails := ResultsDetails{filePath, make([]Details, 0), make([]Details, 0), make([]Details, 0)}
		resultDetails.FailureList = append(resultDetails.FailureList, failureDetails)
		r.Results = append(r.Results, resultDetails)
	}
	r.updateResultsSummary(category)
}

func (r *DetectionResults) Warn(filePath gitrepo.FilePath, category string, message string, commits []string, severity severity.Severity) {
	isFilePresentInResults := false
	for resultIndex := 0; resultIndex < len(r.Results); resultIndex++ {
		if r.Results[resultIndex].Filename == filePath {
			isFilePresentInResults = true
			isEntryPresentForGivenCategoryAndMessage := false
			for detailIndex := 0; detailIndex < len(r.Results[resultIndex].WarningList); detailIndex++ {
				if strings.Compare(r.Results[resultIndex].WarningList[detailIndex].Category, category) == 0 && strings.Compare(r.Results[resultIndex].WarningList[detailIndex].Message, message) == 0 {
					isEntryPresentForGivenCategoryAndMessage = true
					r.Results[resultIndex].WarningList[detailIndex].Commits = append(r.Results[resultIndex].WarningList[detailIndex].Commits, commits...)
				}
			}
			if !isEntryPresentForGivenCategoryAndMessage {
				r.Results[resultIndex].WarningList = append(r.Results[resultIndex].WarningList, Details{category, message, commits, severity})
			}
		}
	}
	if !isFilePresentInResults {
		warningDetails := Details{category, message, commits, severity}
		resultDetails := ResultsDetails{filePath, make([]Details, 0), make([]Details, 0), make([]Details, 0)}
		resultDetails.WarningList = append(resultDetails.WarningList, warningDetails)
		r.Results = append(r.Results, resultDetails)
	}
	r.Summary.Types.Warnings++
}

//Ignore is used to mark the supplied FilePath as being ignored.
//The most common reason for this is that the FilePath is Denied by the Ignores supplied to the Detector, however, Detectors may use more sophisticated reasons to ignore files.
func (r *DetectionResults) Ignore(filePath gitrepo.FilePath, category string) {

	isFilePresentInResults := false
	for resultIndex := 0; resultIndex < len(r.Results); resultIndex++ {
		if r.Results[resultIndex].Filename == filePath {
			isFilePresentInResults = true
			isEntryPresentForGivenCategory := false
			for detailIndex := 0; detailIndex < len(r.Results[resultIndex].IgnoreList); detailIndex++ {
				if strings.Compare(r.Results[resultIndex].IgnoreList[detailIndex].Category, category) == 0 {
					isEntryPresentForGivenCategory = true

				}
			}
			if !isEntryPresentForGivenCategory {
				detail := Details{category, "", make([]string, 0), severity.Low}
				r.Results[resultIndex].IgnoreList = append(r.Results[resultIndex].IgnoreList, detail)
			}
		}
	}
	if !isFilePresentInResults {
		ignoreDetails := Details{category, "", make([]string, 0), severity.Low}
		resultDetails := ResultsDetails{filePath, make([]Details, 0), make([]Details, 0), make([]Details, 0)}
		resultDetails.IgnoreList = append(resultDetails.IgnoreList, ignoreDetails)
		r.Results = append(r.Results, resultDetails)
	}
	r.Summary.Types.Ignores++
}

func (r *DetectionResults) updateResultsSummary(category string) {
	switch category {
	case "filecontent":
		r.Summary.Types.Filecontent++
	case "filename":
		r.Summary.Types.Filename++
	case "filesize":
		r.Summary.Types.Filesize++
	}
}

//HasFailures answers if any Failures were detected for any FilePath in the current run
func (r *DetectionResults) HasFailures() bool {
	return r.Summary.Types.Filesize > 0 || r.Summary.Types.Filename > 0 || r.Summary.Types.Filecontent > 0
}

//HasIgnores answers if any FilePaths were ignored in the current run
func (r *DetectionResults) HasIgnores() bool {
	return r.Summary.Types.Ignores > 0
}

func (r *DetectionResults) HasWarnings() bool {
	return r.Summary.Types.Warnings > 0
}

func (r *DetectionResults) HasDetectionMessages() bool {
	return r.HasWarnings() || r.HasFailures() || r.HasIgnores()
}

//Successful answers if no detector was able to find any possible result to fail the run
func (r *DetectionResults) Successful() bool {
	return !r.HasFailures()
}

//GetFailures returns the various reasons that a given FilePath was marked as failing by all the detectors in the current run
func (r *DetectionResults) GetFailures(fileName gitrepo.FilePath) []Details {
	results := r.getResultDetailsForFilePath(fileName)
	if results == nil {
		return []Details{}
	}
	return results.FailureList
}

func (r *DetectionResults) ReportWarnings() string {
	var result string
	var filePathsForWarnings []string
	var data [][]string

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"File", "Warnings", "Severity"})
	table.SetRowLine(true)

	for _, resultDetails := range r.Results {
		if len(resultDetails.WarningList) > 0 {
			filePathsForWarnings = append(filePathsForWarnings, string(resultDetails.Filename))
			warningData := r.ReportFileWarnings(resultDetails.Filename)
			data = append(data, warningData...)
		}
	}

	filePathsForWarnings = utility.UniqueItems(filePathsForWarnings)
	if r.Summary.Types.Warnings > 0 {
		fmt.Printf("\n\x1b[1m\x1b[31mTalisman Warnings:\x1b[0m\x1b[0m\n")
		table.AppendBulk(data)
		table.Render()
		result = result + fmt.Sprintf("\n\x1b[33mPlease review the above file(s) to make sure that no sensitive content is being pushed\x1b[0m\n")
		result = result + fmt.Sprintf("\n")
	}
	return result
}

//Report returns a string documenting the various failures and ignored files for the current run
func (r *DetectionResults) Report(promptContext prompt.PromptContext) string {
	var result string
	var filePathsForIgnoresAndFailures []string
	var data [][]string

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"File", "Errors", "Severity"})
	table.SetRowLine(true)

	for _, resultDetails := range r.Results {
		if len(resultDetails.FailureList) > 0 || len(resultDetails.IgnoreList) > 0 {
			filePathsForIgnoresAndFailures = append(filePathsForIgnoresAndFailures, string(resultDetails.Filename))
			failureData := r.ReportFileFailures(resultDetails.Filename)
			data = append(data, failureData...)
		}
	}

	filePathsForIgnoresAndFailures = utility.UniqueItems(filePathsForIgnoresAndFailures)

	if r.HasFailures() {
		fmt.Printf("\n\x1b[1m\x1b[31mTalisman Report:\x1b[0m\x1b[0m\n")
		table.AppendBulk(data)
		table.Render()
		fmt.Println()
		r.suggestTalismanRC(filePathsForIgnoresAndFailures, promptContext)
	}
	return result
}

func (r *DetectionResults) suggestTalismanRC(filePaths []string, promptContext prompt.PromptContext) {
	var entriesToAdd []talismanrc.IgnoreConfig

	for _, filePath := range filePaths {
		currentChecksum := utility.DefaultSHA256Hasher{}.CollectiveSHA256Hash([]string{filePath})
		fileIgnoreConfig := talismanrc.BuildIgnoreConfig(r.mode, filePath, currentChecksum, []string{})
		entriesToAdd = append(entriesToAdd, fileIgnoreConfig)
	}

	if promptContext.Interactive && runtime.GOOS != "windows" {
		confirmedEntries := getUserConfirmation(entriesToAdd, promptContext)
		talismanrc.ConfigFromFile().AddIgnores(r.mode, confirmedEntries)
		output, err := exec.Command("git", "add", ".talismanrc").CombinedOutput()
		if err != nil {
			logrus.Errorf("Error appending to talismanrc %v", output)
		}
	} else {
		printTalismanIgnoreSuggestion(entriesToAdd)
		return
	}

}

func getUserConfirmation(configs []talismanrc.IgnoreConfig, promptContext prompt.PromptContext) []talismanrc.IgnoreConfig {
	confirmed := []talismanrc.IgnoreConfig{}
	if len(configs) != 0 {
		fmt.Println("==== Interactively adding to talismanrc ====")
	}
	for _, config := range configs {
		if confirm(config, promptContext) {
			confirmed = append(confirmed, config)
		}
	}
	return confirmed
}

func printTalismanIgnoreSuggestion(entriesToAdd []talismanrc.IgnoreConfig) {
	talismanRC := &talismanrc.TalismanRC{IgnoreConfigs: entriesToAdd}
	ignoreEntries, _ := yaml.Marshal(talismanRC)
	suggestString := fmt.Sprintf("\n\x1b[33mIf you are absolutely sure that you want to ignore the " +
		"above files from talisman detectors, consider pasting the following format in .talismanrc file" +
		" in the project root\x1b[0m\n")
	fmt.Println(suggestString)
	fmt.Println(string(ignoreEntries))
}

func confirm(config talismanrc.IgnoreConfig, promptContext prompt.PromptContext) bool {
	bytes, err := yaml.Marshal(&config)
	if err != nil {
		logrus.Errorf("error marshalling file ignore config: %s", err)
	}

	fmt.Println()
	fmt.Println(string(bytes))

	confirmationString := fmt.Sprintf("Do you want to add %s with above checksum in talismanrc ?", config.GetFileName())

	return promptContext.Prompt.Confirm(confirmationString)
}

//ReportFileFailures adds a string to table documenting the various failures detected on the supplied FilePath by all detectors in the current run
func (r *DetectionResults) ReportFileFailures(filePath gitrepo.FilePath) [][]string {
	failureList := r.getResultDetailsForFilePath(filePath).FailureList
	var data [][]string
	if len(failureList) > 0 {
		for _, detail := range failureList {
			if len(detail.Message) > 150 {
				detail.Message = detail.Message[:75] + "\n" + detail.Message[75:147] + "..."
			}
			data = append(data, []string{string(filePath), detail.Message, detail.Severity.String()})
		}
	}
	return data
}

func (r *DetectionResults) ReportFileWarnings(filePath gitrepo.FilePath) [][]string {
	warningList := r.getResultDetailsForFilePath(filePath).WarningList
	var data [][]string
	if len(warningList) > 0 {
		for _, detail := range warningList {
			if len(detail.Message) > 150 {
				detail.Message = detail.Message[:75] + "\n" + detail.Message[75:147] + "..."
			}
			data = append(data, []string{string(filePath), detail.Message, detail.Severity.String()})
		}
	}
	return data
}
