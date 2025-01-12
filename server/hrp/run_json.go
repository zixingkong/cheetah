package hrp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"github.com/test-instructor/cheetah/server/global"
	"github.com/test-instructor/cheetah/server/hrp/internal/builtin"
	"github.com/test-instructor/cheetah/server/hrp/internal/sdk"
	"github.com/test-instructor/cheetah/server/model/interfacecase"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

func (r *HRPRunner) RunJsons(testcases ...ITestCase) (interfacecase.ApiReport, error) {
	event := sdk.EventTracking{
		Category: "RunAPITests",
		Action:   "hrp run",
	}
	// report start event
	go func() {
		err := sdk.SendEvent(event)
		if err != nil {

		}
	}()
	// report execution timing event
	defer func(e sdk.IEvent) {
		err := sdk.SendEvent(e)
		if err != nil {

		}
	}(event.StartTiming("execution"))
	// record execution data to summary
	s := newOutSummary()

	// load all testcases
	testCases, err := LoadTestCases(testcases...)
	if err != nil {
		log.Error().Err(err).Msg("run json failed to load testcases")
		return interfacecase.ApiReport{}, err
	}

	// run testcase one by one
	for _, testcase := range testCases {
		sessionRunner, err := r.NewSessionRunner(testcase)

		if err != nil {
			log.Error().Err(err).Msg("[Run] init session runner failed")
			return interfacecase.ApiReport{}, err
		}
		defer func() {
			if sessionRunner.parser.plugin != nil {
				sessionRunner.parser.plugin.Quit()
			}
		}()

		for it := sessionRunner.parametersIterator; it.HasNext(); {
			if err = sessionRunner.Start(it.Next()); err != nil {
				log.Error().Err(err).Msg("[Run] run testcase failed")
				return interfacecase.ApiReport{}, err
			}
			caseSummary := sessionRunner.GetSummary()
			caseSummary.CaseID = testcase.ID
			//for k, _ := range caseSummary.Records {
			//	caseSummary.Records[k].ValidatorsNumber = testcase.TestSteps[k].Struct().ValidatorsNumber
			//}
			caseSummary.Name = testcase.Name
			s.appendCaseSummary(caseSummary)
		}
	}
	s.Time.Duration = time.Since(s.Time.StartAt).Seconds()

	// save summary
	if r.saveTests {
		err := s.genSummary()
		if err != nil {
			return interfacecase.ApiReport{}, err
		}
	}

	// generate HTML report
	if r.genHTMLReport {
		err := s.genHTMLReport()
		if err != nil {
			return interfacecase.ApiReport{}, err
		}
	}
	sj, _ := json.Marshal(s)
	global.GVA_LOG.Debug("测试报告json格式")
	global.GVA_LOG.Debug("\n" + string(sj))
	var reportsStruct interfacecase.ApiReport
	err = json.Unmarshal(sj, &reportsStruct)
	return reportsStruct, nil
}

func tmpls(relativePath, debugTalkFileName string) string {
	return filepath.Join(debugTalkFileName, relativePath)
}

func BuildHashicorpPyPlugin(debugTalkByte []byte, debugTalkFilePath string) {
	log.Info().Msg("[init] prepare hashicorp python plugin")
	//src, _ := ioutil.ReadFile(tmpls("plugin/debugtalk.py"))
	err := ioutil.WriteFile(tmpls("debugtalk.py", debugTalkFilePath), debugTalkByte, 0o644)
	if err != nil {
		log.Error().Err(err).Msg("copy hashicorp python plugin failed")
		os.Exit(1)
	}
}

func RemoveHashicorpPyPlugin(debugTalkFilePath string) {
	log.Info().Msg("[teardown] remove hashicorp python plugin")
	// on v4.1^, running case will generate .debugtalk_gen.py used by python plugin
	os.Remove(tmpls(PluginPySourceFile, debugTalkFilePath))
	os.Remove(tmpls(PluginPySourceGenFile, debugTalkFilePath))
}

type TestCaseJson struct {
	JsonString        string
	ID                uint
	DebugTalkFilePath string
	Config            *TConfig
	Name              string
}

func (testCaseJson *TestCaseJson) GetPath() string {
	return testCaseJson.DebugTalkFilePath
}

func (testCaseJson *TestCaseJson) ToTestCase() (*TestCase, error) {
	tc := &TCase{}
	var err error
	//将用例转换成TCase
	casePath := testCaseJson.JsonString
	tc, err = loadFromString(casePath)
	if err != nil {
		return nil, err
	}

	err = tc.MakeCompat()
	if err != nil {
		return nil, err
	}

	tc.Config.Path = testCaseJson.GetPath()
	testCaseJson.Config.Path = testCaseJson.GetPath()

	//将用例转成成TestCase
	testCase := &TestCase{
		ID:     testCaseJson.ID,
		Name:   testCaseJson.Name,
		Config: testCaseJson.Config,
	}

	projectRootDir, err := GetProjectRootDirPath(testCaseJson.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project root dir")
	}

	// load .env file
	dotEnvPath := filepath.Join(projectRootDir, ".env")
	if builtin.IsFilePathExists(dotEnvPath) {
		envVars := make(map[string]string)
		err = builtin.LoadFile(dotEnvPath, envVars)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load .env file")
		}

		// override testcase config env with variables loaded from .env file
		// priority: .env file > testcase config env
		if testCase.Config.Environs == nil {
			testCase.Config.Environs = make(map[string]string)
		}
		for key, value := range envVars {
			testCase.Config.Environs[key] = value
		}
	}

	for _, step := range tc.TestSteps {
		step.ParntID = step.ID
		step.ID = 0
		if step.TestCase != nil {
			testcaseCheetahStr, _ := json.Marshal(step.TestCase)
			apiConfig_json, _ := json.Marshal(step.TestCase.(map[string]interface{})["Config"])
			var tConfig TConfig
			json.Unmarshal(apiConfig_json, &tConfig)
			tcj := &TestCaseJson{
				JsonString:        string(testcaseCheetahStr),
				ID:                step.ID,
				DebugTalkFilePath: testCaseJson.GetPath(),
				Config:            &tConfig,
				Name:              testCase.Name,
			}
			tc, _ := tcj.ToTestCase()
			step.TestCase = tc

			_, ok := step.TestCase.(*TestCase)
			if !ok {
				return nil, fmt.Errorf("failed to handle referenced testcase, got %v", step.TestCase)
			}
			testCase.TestSteps = append(testCase.TestSteps, &StepTestCaseWithOptionalArgs{
				step: step,
			})
		} else if step.ThinkTime != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepThinkTime{
				step: step,
			})
		} else if step.Request != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepRequestWithOptionalArgs{
				step: step,
			})
		} else if step.Transaction != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepTransaction{
				step: step,
			})
		} else if step.Rendezvous != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepRendezvous{
				step: step,
			})
		} else if step.WebSocket != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepWebSocket{
				step: step,
			})
		} else {
			log.Warn().Interface("step", step).Msg("[convertTestCase] unexpected step")
		}
	}
	return testCase, nil
}

func loadFromString(jsonString string) (*TCase, error) {
	tc := &TCase{}
	decoder := json.NewDecoder(bytes.NewReader([]byte(jsonString)))
	decoder.UseNumber()
	err := decoder.Decode(tc)
	return tc, err
}

type JsonToCase struct {
	JsonString        string
	ID                uint
	DebugTalkFilePath string
	Name              string
}

func (testCaseJson *JsonToCase) GetPath() string {
	return testCaseJson.DebugTalkFilePath
}

func (testCaseJson *JsonToCase) ToTestCase() (ITestCase, error) {
	tc := &TCase{}
	var err error
	casePath := testCaseJson.JsonString
	tc, err = loadFromString(casePath)
	if err != nil {
		return nil, err
	}

	err = tc.MakeCompat()
	if err != nil {
		return nil, err
	}

	tc.Config.Path = testCaseJson.GetPath()
	testCase := &TestCase{
		ID:     testCaseJson.ID,
		Name:   testCaseJson.Name,
		Config: tc.Config,
	}

	projectRootDir, err := GetProjectRootDirPath(testCaseJson.GetPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed to get project root dir")
	}

	// load .env file
	dotEnvPath := filepath.Join(projectRootDir, ".env")
	if builtin.IsFilePathExists(dotEnvPath) {
		envVars := make(map[string]string)
		err = builtin.LoadFile(dotEnvPath, envVars)
		if err != nil {
			return nil, errors.Wrap(err, "failed to load .env file")
		}

		// override testcase config env with variables loaded from .env file
		// priority: .env file > testcase config env
		if testCase.Config.Environs == nil {
			testCase.Config.Environs = make(map[string]string)
		}
		for key, value := range envVars {
			testCase.Config.Environs[key] = value
		}
	}

	for _, step := range tc.TestSteps {
		step.ParntID = step.ID
		step.ID = 0
		if step.TestCase != nil {
			caseStr, _ := json.Marshal(step.TestCase)
			jtc := &JsonToCase{
				JsonString: string(caseStr),
				ID:         testCase.ID,
				Name:       testCase.Name,
			}

			tc, err := jtc.ToTestCase()
			if err != nil {
				return nil, err
			}
			step.TestCase = tc
			testCase.TestSteps = append(testCase.TestSteps, &StepTestCaseWithOptionalArgs{
				step: step,
			})
		} else if step.ThinkTime != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepThinkTime{
				step: step,
			})
		} else if step.Request != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepRequestWithOptionalArgs{
				step: step,
			})
		} else if step.Transaction != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepTransaction{
				step: step,
			})
		} else if step.Rendezvous != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepRendezvous{
				step: step,
			})
		} else if step.WebSocket != nil {
			testCase.TestSteps = append(testCase.TestSteps, &StepWebSocket{
				step: step,
			})
		} else {
			log.Warn().Interface("step", step).Msg("[convertTestCase] unexpected step")
		}
	}
	return testCase, nil
}
