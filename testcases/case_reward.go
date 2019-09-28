package testcases

import (
	"context"
	"errors"
	"fmt"
	"github.com/PlatONnetwork/PlatON-Go/common/vm"
	"github.com/PlatONnetwork/PlatON-Go/core/types"
	"github.com/PlatONnetwork/PlatON-Go/x/xcom"
	"log"
	"math/big"
)

type rewardCases struct {
	commonCases
}

func (r *rewardCases) Prepare() error {
	if err := r.commonCases.Prepare(); err != nil {
		return err
	}
	return nil
}

func (r *rewardCases) Start() error {

	return nil
}

func (r *rewardCases) Exec(caseName string) error {
	return r.commonCases.exec(caseName, r)
}

func (r *rewardCases) End() error {
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

func (r *rewardCases) List() []string {
	return r.list(r)
}

func (r *rewardCases) CaseRewardPoolBalance() error {
	ctx := context.Background()
	totalRewardBalance := r.GetBalance(ctx, vm.RewardManagerPoolAddr, big.NewInt(0))
	totalCommunityDeveloper := r.GetBalance(ctx, xcom.CDFAccount(), big.NewInt(0))
	totalPlatONFoundation := r.GetBalance(ctx, xcom.PlatONFundAccount(), big.NewInt(0))
	res := r.CallGetRestrictingInfo(ctx, vm.RewardManagerPoolAddr)
	log.Printf("Restricting plan:%+v", res)

	f := func(block *types.Block, params ...interface{}) (bool, error) {
		year := params[0].(int)
		if block.Number().Int64() >= int64(year*1600) {
			res2 := r.CallGetRestrictingInfo(ctx, vm.RewardManagerPoolAddr)
			log.Printf("block:%v Restricting plan:%+v", block, res2)
			Reward := params[1].(*big.Int)
			Foundation := params[2].(*big.Int)
			Community := params[3].(*big.Int)
			CommunityDeveloperBalance := r.GetBalance(ctx, xcom.CDFAccount(), nil)
			PlatONFoundationBalance := r.GetBalance(ctx, xcom.PlatONFundAccount(), nil)
			rewardBalance := r.GetBalance(ctx, vm.RewardManagerPoolAddr, nil)
			log.Printf("reward: want %v have %v", Reward, rewardBalance)
			log.Printf("CommunityDeveloperBalance: want %v have %v", Community, CommunityDeveloperBalance)
			log.Printf("PlatONFoundationBalance: want %v have %v", Foundation, PlatONFoundationBalance)
			return true, nil
		}
		return false, nil
	}
	totalAmount, _ := new(big.Int).SetString("10250000000000000000000000000", 10)
	for i := 0; i < 12; i++ {
		increse := new(big.Int).Div(totalAmount, big.NewInt(40))
		tmp := new(big.Int).Mul(increse, big.NewInt(80))
		giveReward := new(big.Int).Div(tmp, big.NewInt(100))
		if i < len(res.Entry) {
			totalRewardBalance.Add(totalRewardBalance, res.Entry[i].Amount)
		}
		if i < 10 {
			giveCommunityDeveloper := new(big.Int).Sub(increse, giveReward)
			totalRewardBalance.Add(totalRewardBalance, giveReward)
			totalCommunityDeveloper.Add(totalCommunityDeveloper, giveCommunityDeveloper)
		} else {
			tmp3 := new(big.Int).Sub(increse, giveReward)
			giveCommunityDeveloper := new(big.Int).Div(tmp3, big.NewInt(2))
			givePlatONFoundation := new(big.Int).Div(tmp3, big.NewInt(2))
			totalCommunityDeveloper.Add(totalCommunityDeveloper, giveCommunityDeveloper)
			totalRewardBalance.Add(totalRewardBalance, giveReward)
			totalPlatONFoundation.Add(totalPlatONFoundation, givePlatONFoundation)
		}
		r.addJobs(fmt.Sprintf("cal year %d reward", i), f, i+1, new(big.Int).Set(totalRewardBalance), new(big.Int).Set(totalPlatONFoundation), new(big.Int).Set(totalCommunityDeveloper))

		totalAmount.Add(totalAmount, increse)
	}
	return nil
}

func (r *rewardCases) CaseIncreseBalance() error {
	zeroEpoch := new(big.Int).Mul(big.NewInt(622157424869165), big.NewInt(1e11))
	oneEpoch := new(big.Int).Mul(big.NewInt(559657424869165), big.NewInt(1e11))
	twoEpoch := new(big.Int).Mul(big.NewInt(495594924869165), big.NewInt(1e11))
	threeEpoch := new(big.Int).Mul(big.NewInt(429930862369165), big.NewInt(1e11))
	fourEpoch := new(big.Int).Mul(big.NewInt(362625198306666), big.NewInt(1e11))
	fiveEpoch := new(big.Int).Mul(big.NewInt(293636892642633), big.NewInt(1e11))
	sixEpoch := new(big.Int).Mul(big.NewInt(222923879336939), big.NewInt(1e11))
	sevenEpoch := new(big.Int).Mul(big.NewInt(150443040698633), big.NewInt(1e11))
	eightEpoch := new(big.Int).Mul(big.NewInt(761501810943690), big.NewInt(1e10))
	var plans []*big.Int = []*big.Int{zeroEpoch, oneEpoch, twoEpoch, threeEpoch, fourEpoch, fiveEpoch, sixEpoch, sevenEpoch, eightEpoch}

	rewardBalance, _ := new(big.Int).SetString("200000000000000000000000000", 10)
	PlatONFoundationAddress, _ := new(big.Int).SetString("2000000000000000000000000000", 10)

	rewardBalance.Add(rewardBalance, zeroEpoch)
	log.Print(rewardBalance)
	for _, value := range plans {
		PlatONFoundationAddress.Sub(PlatONFoundationAddress, value)
	}

	totalCommunityDeveloper := new(big.Int)
	totalAmount, _ := new(big.Int).SetString("10250000000000000000000000000", 10)

	stakingAmoount, _ := new(big.Int).SetString("10000000000000000000000000", 10)
	stakingAmoount.Mul(stakingAmoount, big.NewInt(4))

	PlatONFoundationAddress.Sub(PlatONFoundationAddress, stakingAmoount)

	for i := 1; i < 15; i++ {
		increse := new(big.Int).Mul(totalAmount, big.NewInt(25))
		increse.Div(increse, big.NewInt(1000))
		tmp := new(big.Int).Mul(increse, big.NewInt(80))
		log.Print(tmp)
		giveReward := new(big.Int).Div(tmp, big.NewInt(100))
		if i < 9 {
			rewardBalance.Add(rewardBalance, plans[i])
		}
		if i < 9 {
			giveCommunityDeveloper := new(big.Int).Sub(increse, giveReward)
			totalCommunityDeveloper.Add(totalCommunityDeveloper, giveCommunityDeveloper)
		} else {
			giveOther := new(big.Int).Sub(increse, giveReward)
			tmp := new(big.Int).Mul(giveOther, big.NewInt(5))
			tmp2 := new(big.Int).Div(tmp, big.NewInt(10))
			PlatONFoundationAddress.Add(PlatONFoundationAddress, tmp2)
			totalCommunityDeveloper.Add(totalCommunityDeveloper, tmp2)
		}
		rewardBalance.Add(rewardBalance, giveReward)
		totalAmount.Add(totalAmount, increse)
		log.Printf("year %v increase:%v totalAmount %v giveReward:%v  rewardBalance %v CommunityDeveloper %v PlatONFoundation%v", i, increse, totalAmount, giveReward, rewardBalance, totalCommunityDeveloper, PlatONFoundationAddress)
	}
	return nil
}

func (r *rewardCases) CaseCal() error {
	b, _ := new(big.Int).SetString("262215742486916500000000000", 10)
	b.Div(b, big.NewInt(2))
	b.Div(b, big.NewInt(10))
	log.Print(b)
	b.Div(b, big.NewInt(4))
	log.Print(b)
	return nil
}
