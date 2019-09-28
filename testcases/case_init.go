package testcases

import (
	"context"
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/vm"
	"github.com/PlatONnetwork/PlatON-Go/x/xcom"
	"log"
	"math/big"
	"reflect"
	"strings"
)

type initCases struct {
	commonCases
	cxt context.Context
}

func (r *initCases) Prepare() error {
	if err := r.commonCases.Prepare(); err != nil {
		return err
	}
	ctx := context.Background()
	r.cxt = ctx
	return nil
}

func (r *initCases) Start() error {
	object := reflect.TypeOf(r)
	for i := 0; i < object.NumMethod(); i++ {
		method := object.Method(i)
		if strings.HasPrefix(method.Name, PrefixCase) {
			val := reflect.ValueOf(r).MethodByName(method.Name).Call([]reflect.Value{})
			if !val[0].IsNil() {
				err := val[0].Interface().(error)
				return err
			}
		}
	}
	return nil
}

func (r *initCases) Exec(caseName string) error {
	return r.commonCases.exec(caseName, r)
}

func (r *initCases) End() error {
	if err := r.commonCases.End(); err != nil {
		return err
	}
	return nil
}

func (r *initCases) List() []string {
	return r.commonCases.list(r)
}

/*
2019/08/31 16:53:27 PlatONFoundation balance:1638688017094410000000000000
2019/08/31 16:53:27 RewardManagerPool balance:262215742486916500000000000
2019/08/31 16:53:27 StakingContract balance:40000000000000000000000000
2019/08/31 16:53:27 RestrictingContract balance:259096240418673500000000000
2019/08/31 16:53:27 0xf66CB3C7f28D058AE3C6eD9493C6A9e2a7d7786d balance:8050000000000000000000000000
*/
func (r *initCases) CaseInitAccountBalance() error {

	balance := r.GetBalance(r.cxt, xcom.PlatONFundAccount(), big.NewInt(1))
	log.Print("PlatONFoundation balance:", balance)
	balance2 := r.GetBalance(r.cxt, vm.RewardManagerPoolAddr, big.NewInt(1))
	log.Print("RewardManagerPool balance:", balance2)
	balance3 := r.GetBalance(r.cxt, vm.StakingContractAddr, big.NewInt(1))
	log.Print("StakingContract balance:", balance3)
	balance4 := r.GetBalance(r.cxt, vm.RestrictingContractAddr, big.NewInt(1))
	log.Print("RestrictingContract balance:", balance4)
	balance5 := r.GetBalance(r.cxt, common.HexToAddress("0xf66CB3C7f28D058AE3C6eD9493C6A9e2a7d7786d"), big.NewInt(1))
	log.Print("0xf66CB3C7f28D058AE3C6eD9493C6A9e2a7d7786d balance:", balance5)
	return nil
}

func (r *initCases) CaseRewardManagerPoolRestrictingRecord() error {
	res := r.CallGetRestrictingInfo(r.cxt, vm.RewardManagerPoolAddr)
	log.Printf("RewardManagerPoolAddr  RestrictingInfo  %+v", res)
	totalMount := new(big.Int)
	for _, value := range res.Entry {
		totalMount.Add(totalMount, (*big.Int)(value.Amount))
	}
	t, _ := new(big.Int).SetString("259096240418673500000000000", 10)
	if totalMount.Cmp(t) != 0 {
		return fmt.Errorf("RewardPool init Restricting Record amount is wrong,want %v have %v", t, totalMount)
	}
	return nil
}
