package testcases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"path"

	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/vm"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/crypto/bls"
	"github.com/PlatONnetwork/PlatON-Go/p2p/discover"
	"github.com/PlatONnetwork/PlatON-Go/x/restricting"
)

type restrictCases struct {
	commonCases
	paramsPath string
	params     restrictingParams
}

type restrictingParams struct {
	CreatePlan       []restricting.RestrictingPlan `json:"create_plan"`
	CasePledgeReturn casePledgeReturnConfig        `json:"case_pledge_return"`
	CreateWrongPlan  []restricting.RestrictingPlan `json:"create_wrong_plan"`
}

type casePledgeReturnConfig struct {
	Staking struct {
		BlsKey string `json:"bls_key"`
		NodeId string `json:"node_id"`
	} `json:"staking"`
}

func (r *restrictCases) Prepare() error {
	if err := r.commonCases.Prepare(); err != nil {
		return err
	}
	r.paramsPath = path.Join(config.Dir, config.RestrictingConfigFile)
	r.parseConfigJson()
	return nil
}

func (r *restrictCases) Start() error {
	if err := r.CaseCreatePlan(); err != nil {
		return r.SendError("restricting createPlan", err)
	}
	if err := r.CasePledgeLockAndReturn(); err != nil {
		return r.SendError("restricting CasePledgeLockAndReturn", err)
	}
	return nil
}

func (r *restrictCases) Exec(caseName string) error {
	return r.commonCases.exec(caseName, r)
}

func (r *restrictCases) End() error {
	if err := r.commonCases.End(); err != nil {
		return err
	}
	if len(r.errors) != 0 {
		for _, value := range r.errors {
			r.SendError("restricting", value)
		}
		return errors.New("run restrictCases fail")
	}
	return nil
}

func (r *restrictCases) List() []string {
	return r.list(r)
}

func (r *restrictCases) parseConfigJson() {
	bytes, err := ioutil.ReadFile(r.paramsPath)
	if err != nil {
		panic(fmt.Errorf("parse restrictCases config file error,%s", err.Error()))
	}
	if err := json.Unmarshal(bytes, &r.params); err != nil {
		panic(fmt.Errorf("parse restrictCases config to json error,%s", err.Error()))
	}
}

func (r *restrictCases) CaseCreateWrongPlan() error {
	ctx := context.Background()
	account, _ := AccountPool.Get().(*PriAccount)
	defer AccountPool.Put(account)
	plans := r.params.CreateWrongPlan
	_, to := r.generateEmptyAccount()
	txHash, err := r.CreateRestrictingPlanTransaction(ctx, account, to, plans)
	if err != nil {
		return err
	}
	log.Print("txHash:", txHash.String())
	if err := r.WaitTransactionByHash(ctx, txHash); err != nil {
		return fmt.Errorf("wait Transaction fail:%v", err)
	}
	res, err := r.GetXcomResult(ctx, txHash)
	if err != nil {
		return err
	}
	log.Print(res.ErrMsg)
	result := r.CallGetRestrictingInfo(ctx, to)
	log.Printf("plans: %+v", result)
	log.Print("begin next RestrictingPlan")
	txHash2, err := r.CreateRestrictingPlanTransaction(ctx, account, to, plans)
	if err != nil {
		return err
	}
	log.Print("txHash2:", txHash2.String())
	if err := r.WaitTransactionByHash(ctx, txHash2); err != nil {
		return fmt.Errorf("wait Transaction fail:%v", err)
	}
	res2, err := r.GetXcomResult(ctx, txHash2)
	if err != nil {
		return err
	}
	log.Print(res2.ErrMsg)
	result2 := r.CallGetRestrictingInfo(ctx, to)
	log.Printf("plans2: %+v", result2)

	return nil
}

func (r *restrictCases) CaseCreatePlan() error {
	ctx := context.Background()
	account, _ := AccountPool.Get().(*PriAccount)
	defer AccountPool.Put(account)
	plans := r.params.CreatePlan
	totalAmount := new(big.Int)
	for _, value := range plans {
		totalAmount.Add(totalAmount, value.Amount)
	}
	_, to := r.generateEmptyAccount()

	oldRestrictingContractAddr := r.GetBalance(ctx, vm.RestrictingContractAddr, nil)
	oldTo := r.GetBalance(ctx, to, nil)
	oldFrom := r.GetBalance(ctx, account.Address, nil)
	txHash, err := r.CreateRestrictingPlanTransaction(ctx, account, to, plans)
	if err != nil {
		return err
	}
	if err := r.WaitTransactionByHash(ctx, txHash); err != nil {
		return fmt.Errorf("wait Transaction fail:%v", err)
	}

	//balance on RestrictingContractAddr
	newRestricting := r.GetBalance(ctx, vm.RestrictingContractAddr, nil)

	if new(big.Int).Sub(newRestricting, oldRestrictingContractAddr).Cmp(totalAmount) != 0 {
		return fmt.Errorf("RestrictingContractAddr balance is wrong,want %v,have %v", new(big.Int).Sub(newRestricting, oldRestrictingContractAddr), totalAmount)
	}

	tx, _, err := r.client.TransactionByHash(ctx, txHash)
	if err != nil {
		return err
	}

	totalGasAmountUsed := new(big.Int).Mul(tx.GasPrice(), new(big.Int).SetInt64(int64(tx.Gas())))
	tmp2 := new(big.Int).Add(totalAmount, totalGasAmountUsed)

	//balance on from
	newFrom := r.GetBalance(ctx, account.Address, nil)
	if new(big.Int).Sub(oldFrom, newFrom).Cmp(tmp2) != 0 {
		return fmt.Errorf("from account %v  balance is wrong,want %v,have %v", account.Address.String(), tmp2, new(big.Int).Sub(oldFrom, newFrom))
	}
	//balance on to
	newTo := r.GetBalance(ctx, to, nil)
	if newTo.Cmp(oldTo) != 0 {
		return fmt.Errorf("to account %v balance is wrong,want %v,have %v", to.String(), oldTo, newTo)
	}
	result := r.CallGetRestrictingInfo(ctx, to)
	log.Printf("plans: %+v", result)
	resAmount := new(big.Int)
	cmpfunc := func(block *types.Block, params ...interface{}) (bool, error) {
		height := params[0].(uint64)
		amount := params[1].(*big.Int)
		if block.Number().Uint64() > height {
			balance := r.GetBalance(ctx, to, nil)
			if balance.Cmp(amount) != 0 {
				return false, fmt.Errorf("amount not comprare,want %v,have %v", amount, balance)
			}
			resBalance := r.GetBalance(ctx, vm.RestrictingContractAddr, nil)
			if new(big.Int).Sub(newRestricting, resBalance).Cmp(amount) != 0 {
				return false, fmt.Errorf("RestrictingContractAddr amount not comprare,want %v,have %v", amount, new(big.Int).Sub(newRestricting, resBalance))
			}
			return true, nil
		}
		return false, nil
	}
	for _, val := range result.Entry {
		tmp := new(big.Int).Add(resAmount, val.Amount)
		resAmount.Add(resAmount, val.Amount)
		r.addJobs(fmt.Sprintf("锁仓释放计划高度:%v", val.Height), cmpfunc, val.Height, tmp)
	}
	return nil
}

func (r *restrictCases) CasePledgeLockAndReturn() error {
	ctx := context.Background()
	_, err := r.GetProgramVersion(ctx)
	if err != nil {
		return err
	}
	stakingAccount, _ := AccountPool.Get().(*PriAccount)

	versionValue, err := r.GetProgramVersion(ctx)
	if err != nil {
		return err
	}

	proof, err := r.GetSchnorrNIZKProve(ctx)
	if err != nil {
		return err
	}

	var input stakingInput

	// programVersion
	input.ProgramVersion = versionValue.Version
	// programVersionSign
	var versionSign common.VersionSign
	versionSign.UnmarshalText([]byte(versionValue.Sign))
	input.ProgramVersionSign = versionSign

	// bls publicKey
	var keyHex bls.PublicKeyHex
	keyHex.UnmarshalText([]byte(r.params.CasePledgeReturn.Staking.BlsKey))
	input.BlsPubKey = keyHex
	// bls proof
	var proofHex bls.SchnorrProofHex
	proofHex.UnmarshalText([]byte(proof))
	input.BlsProof = proofHex

	input.Amount, _ = new(big.Int).SetString("10000000000000000000000000", 10)
	input.Typ = 0
	_, add := r.generateEmptyAccount()
	input.BenefitAddress = add
	id, err := discover.HexID(r.params.CasePledgeReturn.Staking.NodeId)
	if err != nil {
		return err
	}
	input.NodeId = id

	log.Print("begin create staking")
	txhash2, err := r.CreateStakingTransaction(ctx, stakingAccount, input)
	if err != nil {
		return fmt.Errorf("createStakingTransaction fail:%v", err)
	}
	AccountPool.Put(stakingAccount)

	if err := r.WaitTransactionByHash(ctx, txhash2); err != nil {
		return fmt.Errorf("wait Transaction2 %v fail:%v", txhash2, err)
	}
	log.Print("end create staking", txhash2.String())

	xres, err := r.GetXcomResult(ctx, txhash2)
	if err != nil {
		return err
	}
	log.Printf("end create staking,res:%+v", xres)

	log.Print("begin create restricting plans")
	createRestrictingAccount, _ := AccountPool.Get().(*PriAccount)
	plans := make([]restricting.RestrictingPlan, 0)

	amount, _ := new(big.Int).SetString("1000000000000000000000000", 10)
	plans = append(plans, restricting.RestrictingPlan{1, amount})
	getRestrictingAccount, _ := AccountPool.Get().(*PriAccount)
	txHash, err := r.CreateRestrictingPlanTransaction(ctx, createRestrictingAccount, getRestrictingAccount.Address, plans)
	if err != nil {
		return fmt.Errorf("CreateRestrictingPlanTransaction fail:%v", err)
	}
	AccountPool.Put(createRestrictingAccount)

	if err := r.WaitTransactionByHash(ctx, txHash); err != nil {
		return fmt.Errorf("wait Transaction %v fail:%v", txHash, err)
	}

	amount2, _ := new(big.Int).SetString("1000000000000000000000000", 10)
	log.Print("begin delegateTransaction", amount2)

	txhash4, err := r.DelegateTransaction(ctx, getRestrictingAccount, id, 1, amount2)
	if err != nil {
		return fmt.Errorf("delegateTransaction fail:%v", err)
	}
	if err := r.WaitTransactionByHash(ctx, txhash4); err != nil {
		return fmt.Errorf("wait Transaction %v fail:%v", txhash4, err)
	}

	log.Print("end delegateTransaction", txhash4.String())

	xres2, err := r.GetXcomResult(ctx, txhash4)
	if err != nil {
		return err
	}
	log.Printf("end DelegateTransaction staking,res:%+v", xres2)

	res := r.CallGetRestrictingInfo(ctx, getRestrictingAccount.Address)
	log.Printf("RestrictingInfo:%+v", res)
	quene, err := r.CallGetRelatedListByDelAddr(ctx, getRestrictingAccount.Address)
	if err != nil {
		return fmt.Errorf("getRelatedListByDelAddr fail:%v", err)
	}
	log.Print("getRelatedListByDelAddr", quene)
	tmpfunc := func(block *types.Block, params ...interface{}) (bool, error) {
		hight := params[0].(uint64)
		if block.Number().Uint64() > hight {
			balance := params[1].(*big.Int)
			txhash, err := r.WithdrewDelegateTransaction(ctx, quene[0].StakingBlockNum, id, getRestrictingAccount, balance)
			if err != nil {
				return false, err
			}
			if err := r.WaitTransactionByHash(ctx, txhash); err != nil {
				return false, err
			}
			result := r.CallGetRestrictingInfo(ctx, getRestrictingAccount.Address)
			log.Printf("plans: %+v", result)
			return true, nil
		}
		return false, nil
	}
	releaseAccount := func(block *types.Block, params ...interface{}) (bool, error) {
		hight := params[0].(uint64)
		if block.Number().Uint64() > hight {
			AccountPool.Put(getRestrictingAccount)
		}
		return false, nil
	}
	amount3, _ := new(big.Int).SetString("5000000000000000000000000", 10)
	r.addJobs("锁仓转质押1是否被释放", tmpfunc, res.Entry[0].Height, amount3)
	r.addJobs("锁仓转质押2是否被释放", tmpfunc, res.Entry[0].Height+300, amount3)
	r.addJobs("释放锁仓账户", releaseAccount, res.Entry[0].Height+310)

	return nil
}
