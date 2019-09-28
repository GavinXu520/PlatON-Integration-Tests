package testcases

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"fmt"
	ethereum "github.com/PlatONnetwork/PlatON-Go"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/vm"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/crypto"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
	"github.com/PlatONnetwork/PlatON-Go/p2p/discover"
	"github.com/PlatONnetwork/PlatON-Go/rlp"
	"github.com/PlatONnetwork/PlatON-Go/x/restricting"
	"github.com/PlatONnetwork/PlatON-Go/x/staking"
	"github.com/PlatONnetwork/PlatON-Go/x/xcom"
	"github.com/robfig/cron"
	"log"
	"math/big"
	"reflect"
	"strings"
	"time"
)

const PrefixCase = "Case"

type commonCases struct {
	client *ethclient.Client
	//adrs   []common.Address
	corn   *cron.Cron
	jobs   []*job
	errors []error
	errch  chan error
	donech chan struct{}
}

type job struct {
	handle func(block *types.Block, params ...interface{}) (bool, error)
	params []interface{}
	done   bool
	runing bool
	desc   string
}

func (j *job) run(block *types.Block) (bool, error) {
	return j.handle(block, j.params...)
}

func (c *commonCases) list(m caseTest) []string {
	var names []string
	object := reflect.TypeOf(m)
	for i := 0; i < object.NumMethod(); i++ {
		method := object.Method(i)
		if strings.HasPrefix(method.Name, PrefixCase) {
			names = append(names, strings.TrimPrefix(method.Name, "Case"))
		}
	}
	return names
}

func (c *commonCases) exec(caseName string, m caseTest) error {
	methodNames := PrefixCase + caseName
	val := reflect.ValueOf(m).MethodByName(methodNames).Call([]reflect.Value{})
	if val[0].IsNil() {
		return nil
	}
	err := val[0].Interface().(error)
	return err
}

//初始化case
func (c *commonCases) Prepare() error {
	client, err := ethclient.Dial(config.Url)
	if err != nil {
		return err
	}
	c.client = client
	//	c.adrs = GetAllAddress()
	c.corn = cron.New()
	if err := c.corn.AddFunc("@every 1s", c.schedule); err != nil {
		return err
	}
	c.donech = make(chan struct{}, 0)
	c.errch = make(chan error, 0)
	c.errors = make([]error, 0)
	return nil
}

func (c *commonCases) Start() error {
	return nil
}

func (c *commonCases) End() error {
	c.corn.Start()
	for {
		select {
		case err := <-c.errch:
			c.errors = append(c.errors, err)
		case <-c.donech:
			c.corn.Stop()
			close(c.errch)
			return nil
		}
	}
}

func (c *commonCases) GetNonceAt(ctx context.Context, account common.Address, blockNumber *big.Int) (uint64, error) {
	return c.client.NonceAt(ctx, account, blockNumber)
}

func (c *commonCases) encodePPOS(params [][]byte) []byte {
	buf := new(bytes.Buffer)
	err := rlp.Encode(buf, params)
	if err != nil {
		panic(fmt.Errorf("encode rlp data fail: %v", err))
	}
	return buf.Bytes()
}

//等待交易确认
func (c *commonCases) WaitTransactionByHash(ctx context.Context, txHash common.Hash) error {
	i := 0
	for {
		if i > 10 {
			return errors.New("wait to long")
		}
		_, isPending, err := c.client.TransactionByHash(ctx, txHash)
		if err == nil {
			if !isPending {
				break
			}
		} else {
			if err != ethereum.NotFound {
				return err
			}
		}
		time.Sleep(time.Second)
		i++
	}
	return nil
}

type ProgramVersionValue struct {
	ProgramVersion     uint32             `json:"ProgramVersion"`
	ProgramVersionSign common.VersionSign `json:"ProgramVersionSign"`
}

//获取ProgramVersion
func (c *commonCases) CallProgramVersion(ctx context.Context) (*ProgramVersionValue, error) {
	var msg ethereum.CallMsg
	msg.To = &vm.GovContractAddr

	var params [][]byte
	fnType, _ := rlp.EncodeToBytes(uint16(2104))
	params = append(params, fnType)

	send := c.encodePPOS(params)

	msg.Data = send
	data, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	var xres xcom.Result
	if err := json.Unmarshal(data, &xres); err != nil {
		panic(err)
	}
	var VersionValue ProgramVersionValue
	if err := json.Unmarshal([]byte(xres.Data), &VersionValue); err != nil {
		log.Print(xres)
		panic(err)
	}
	return &VersionValue, nil
}

type stakingInput struct {
	Typ            uint16
	BenefitAddress common.Address
	NodeId         discover.NodeID
	ExternalId     string
	NodeName       string
	Website        string
	Details        string
	Amount         *big.Int
	BlsPubKey      string
}

//创建质押
func (c *commonCases) CreateStakingTransaction(ctx context.Context, from *PriAccount, input stakingInput, VersionValue *ProgramVersionValue) (common.Hash, error) {
	var params [][]byte
	params = make([][]byte, 0)
	fnType, _ := rlp.EncodeToBytes(uint16(1000))
	typ, _ := rlp.EncodeToBytes(input.Typ)
	benefitAddress, _ := rlp.EncodeToBytes(input.BenefitAddress)
	nodeId, _ := rlp.EncodeToBytes(input.NodeId)
	externalId, _ := rlp.EncodeToBytes(input.ExternalId)
	nodeName, _ := rlp.EncodeToBytes(input.NodeName)
	website, _ := rlp.EncodeToBytes(input.Website)
	details, _ := rlp.EncodeToBytes(input.Details)
	amount, _ := rlp.EncodeToBytes(input.Amount)
	programVersion, _ := rlp.EncodeToBytes(VersionValue.ProgramVersion)

	programVersionSign, _ := rlp.EncodeToBytes(VersionValue.ProgramVersionSign)
	blsPubKey, _ := rlp.EncodeToBytes(input.BlsPubKey)
	params = append(params, fnType)
	params = append(params, typ)
	params = append(params, benefitAddress)
	params = append(params, nodeId)
	params = append(params, externalId)
	params = append(params, nodeName)
	params = append(params, website)
	params = append(params, details)
	params = append(params, amount)
	params = append(params, programVersion)
	params = append(params, programVersionSign)
	params = append(params, blsPubKey)

	send := c.encodePPOS(params)

	txhash, err := SendRawTransaction(ctx, c.client, from, vm.StakingContractAddr, "0", send)
	if err != nil {
		return common.ZeroHash, err
	}
	return txhash, nil
}

//创建锁仓
func (c *commonCases) CreateRestrictingPlanTransaction(ctx context.Context, from *PriAccount, to common.Address, plans []restricting.RestrictingPlan) (common.Hash, error) {
	var params [][]byte
	params = make([][]byte, 0)
	fnType, _ := rlp.EncodeToBytes(uint16(4000))
	params = append(params, fnType)
	account, _ := rlp.EncodeToBytes(to)
	plansByte, _ := rlp.EncodeToBytes(plans)
	params = append(params, account)
	params = append(params, plansByte)
	send := c.encodePPOS(params)
	txhash, err := SendRawTransaction(ctx, c.client, from, vm.RestrictingContractAddr, "0", send)
	if err != nil {
		return common.ZeroHash, err
	}
	return txhash, nil
}

//创建委托
func (c *commonCases) DelegateTransaction(ctx context.Context, account *PriAccount, nodeID discover.NodeID, typ uint16, amount *big.Int) (common.Hash, error) {
	fnType, _ := rlp.EncodeToBytes(uint16(1004))
	encodeTyp, _ := rlp.EncodeToBytes(typ)
	id, _ := rlp.EncodeToBytes(nodeID)
	encodeAmount, _ := rlp.EncodeToBytes(amount)

	var params [][]byte
	params = append(params, fnType)
	params = append(params, encodeTyp)
	params = append(params, id)
	params = append(params, encodeAmount)
	send := c.encodePPOS(params)

	txhash, err := SendRawTransaction(ctx, c.client, account, vm.StakingContractAddr, "0", send)
	if err != nil {
		return common.ZeroHash, err
	}
	return txhash, nil
}

//查询当前账户地址所委托的节点的NodeID和质押Id
func (c *commonCases) CallGetRelatedListByDelAddr(ctx context.Context, account common.Address) (staking.DelRelatedQueue, error) {
	var msg ethereum.CallMsg
	msg.To = &vm.StakingContractAddr

	var params [][]byte
	fnType, _ := rlp.EncodeToBytes(uint16(1103))
	add, _ := rlp.EncodeToBytes(account)
	params = append(params, fnType)
	params = append(params, add)
	send := c.encodePPOS(params)

	msg.Data = send
	data, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		return nil, err
	}
	var xres xcom.Result
	if err := json.Unmarshal(data, &xres); err != nil {
		panic(err)
	}
	var delRelatedQueue staking.DelRelatedQueue
	if err := json.Unmarshal([]byte(xres.Data), &delRelatedQueue); err != nil {
		log.Print(xres)
		panic(err)
	}
	return delRelatedQueue, nil
}

//减持/撤销委托
func (c *commonCases) WithdrewDelegateTransaction(ctx context.Context, stakingBlockNum uint64, nodeID discover.NodeID, account *PriAccount, amount *big.Int) (common.Hash, error) {
	fnType, _ := rlp.EncodeToBytes(uint16(1005))
	stakingNum, _ := rlp.EncodeToBytes(stakingBlockNum)
	noid, _ := rlp.EncodeToBytes(nodeID)
	amount10, _ := rlp.EncodeToBytes(amount)
	var params [][]byte
	params = append(params, fnType)
	params = append(params, stakingNum)
	params = append(params, noid)
	params = append(params, amount10)
	send := c.encodePPOS(params)

	txhash, err := SendRawTransaction(ctx, c.client, account, vm.StakingContractAddr, "0", send)
	if err != nil {
		return common.ZeroHash, err
	}
	return txhash, nil
}

//获取锁仓信息
func (c *commonCases) CallGetRestrictingInfo(ctx context.Context, account common.Address) restricting.Result {
	var params [][]byte
	params = make([][]byte, 0)
	fnType, _ := rlp.EncodeToBytes(uint16(4100))
	params = append(params, fnType)
	accountBytes, err := rlp.EncodeToBytes(account)
	if err != nil {
		panic(err)
	}
	params = append(params, accountBytes)
	send := c.encodePPOS(params)

	var msg ethereum.CallMsg
	msg.Data = send
	msg.To = &vm.RestrictingContractAddr
	res, err := c.client.CallContract(ctx, msg, nil)
	if err != nil {
		panic(err)
	}
	var xres xcom.Result
	if err := json.Unmarshal(res, &xres); err != nil {
		panic(err)
	}
	if xres.Code != 0 {
		var result restricting.Result
		if err := json.Unmarshal([]byte(xres.Data), &result); err != nil {
			log.Print(xres)
			panic(err)
		}
		return result
	} else {
		return restricting.Result{}
	}
}

//获取账户金额
func (c *commonCases) GetBalance(ctx context.Context, address common.Address, blockNumber *big.Int) *big.Int {
	balance, err := c.client.BalanceAt(ctx, address, blockNumber)
	if err != nil {
		panic(err)
	}
	return balance
}

func (c *commonCases) SendError(caseName string, err error) error {
	log.Printf("[fail]test case %v fail: %v ", caseName, err)
	return fmt.Errorf("test case %v fail: %v ", caseName, err)
}

//添加需等待的测试，每隔1s将当前块高传入f中并执行
// f为需要执行的函数，当执行成功返回true，nil，当未达到执行条件返回false，nil，当执行失败返回false，error
// params为f所需要的参数
func (c *commonCases) addJobs(desc string, f func(block *types.Block, params ...interface{}) (bool, error), params ...interface{}) {
	var j *job
	j = new(job)
	j.params = params
	j.handle = f
	j.desc = desc
	c.jobs = append(c.jobs, j)
}

//生成一个新的空账户，私钥和地址
func (c *commonCases) generateEmptyAccount() (*ecdsa.PrivateKey, common.Address) {
	privateKey, _ := crypto.GenerateKey()
	address := crypto.PubkeyToAddress(privateKey.PublicKey)
	var pri PriAccount
	pri.Priv = privateKey
	pri.Address = address
	allAccounts[address] = &pri
	return privateKey, address
}

func (c *commonCases) schedule() {
	block, err := c.client.BlockByNumber(context.Background(), nil)
	if err != nil {
		panic(err)
	}
	if block.Number().Uint64()%10 == 0 {
		log.Printf("schedule working,current block num is %v", block.Number())
	}
	for i := 0; i < len(c.jobs); {
		if c.jobs[i].done {
			log.Printf("[job] %v done", c.jobs[i].desc)
			c.jobs = append(c.jobs[:i], c.jobs[i+1:]...)
		} else {
			if !c.jobs[i].runing {
				go func(j *job) {
					defer func() {
						if r := recover(); r != nil {
							log.Printf("Recovered in :%v", r)
							c.errch <- fmt.Errorf("job %v painc", j.desc)
							j.done = true
						}
					}()
					j.runing = true
					ok, err := j.run(block)
					if err != nil {
						c.errch <- fmt.Errorf("job %v fail:%v", c.jobs[i].desc, err)
						j.done = true
						return
					}
					if ok {
						j.done = true
					}
					j.runing = false
				}(c.jobs[i])
			}
			i++
		}
	}
	if len(c.jobs) == 0 {
		c.donech <- struct{}{}
		close(c.donech)
	}
}
