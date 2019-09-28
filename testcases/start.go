package testcases

import (
	"fmt"
	"gopkg.in/urfave/cli.v1"
	"log"
	"strings"
	"sync"
)

var (
	StartCmd = cli.Command{
		Name:   "start",
		Usage:  "start all test cases",
		Action: start,
		Flags:  StartFlag,
	}
	ExecCmd = cli.Command{
		Name:   "exec",
		Usage:  "exec a test cases",
		Action: exec,
		Flags:  ExecFlag,
	}
	ListCmd = cli.Command{
		Name:   "list",
		Usage:  "list  cases",
		Action: list,
	}
)

func init() {
	allCases = make(map[string]caseTest)
	allCases["restricting"] = new(restrictCases)
	allCases["init_token"] = new(initCases)
	allCases["reward"] = new(rewardCases)
	allCases["staking"] = new(stakingCases)
}

type caseTest interface {
	Start() error
	Exec(string) error
	Prepare() error
	End() error
	List() []string
}

var allCases map[string]caseTest

func start(c *cli.Context) {
	var wg sync.WaitGroup
	loadData(c.String(ConfigPathFlag.Name))
	for _, value := range allAccounts {
		AccountPool.Put(value)
	}
	for name, value := range allCases {
		wg.Add(1)
		go func(caseFunc caseTest) {
			defer wg.Done()
			if err := caseFunc.Prepare(); err != nil {
				log.Printf("exec %v, Prepare fail:%v", name, err)
				return
			}
			if err := caseFunc.Start(); err != nil {
				log.Printf("exec %v, Start fail:%v", name, err)
				return
			}
			if err := caseFunc.End(); err != nil {
				log.Printf("exec %v, End fail:%v", name, err)
				return
			}
		}(value)
	}
	wg.Wait()
	log.Print("all case exec done")
}

func exec(c *cli.Context) {
	loadData(c.String(ConfigPathFlag.Name))

	funcName := c.String(FuncNameFlag.Name)
	caseName := c.Args().First()

	cases, ok := allCases[caseName]
	if !ok {
		log.Printf("not find the case:%v", caseName)
		return
	}
	if err := cases.Prepare(); err != nil {
		log.Printf("exec %v, Prepare fail:%v", caseName, err)
		return
	}
	if funcName == "" {
		if err := cases.Start(); err != nil {
			log.Printf("exec %v, Start fail:%v", caseName, err)
			return
		}
	} else {
		if err := cases.Exec(funcName); err != nil {
			log.Printf("exec %v, Start fail:%v", funcName, err)
			return
		}
	}
	if err := cases.End(); err != nil {
		log.Printf("exec %v, End fail", caseName)
		return
	}
}

func list(c *cli.Context) {
	caseName := c.Args().First()
	var output string
	if caseName != "" {
		cases := allCases[caseName]
		names := cases.List()
		output = strings.Join(names, ",")
		fmt.Printf("case %s support funcs: %s\n", caseName, output)
	} else {
		var names []string
		for key, _ := range allCases {
			names = append(names, key)
		}
		output = strings.Join(names, ",")
		fmt.Printf("support cases:%s\n", output)
	}
}
