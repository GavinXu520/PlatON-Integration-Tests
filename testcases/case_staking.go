package testcases

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"math/rand"
	"path"
	"sync"
	"time"

	"github.com/PlatONnetwork/PlatON-Go/x/xcom"

	platON "github.com/PlatONnetwork/PlatON-Go"
	"github.com/PlatONnetwork/PlatON-Go/common"
	"github.com/PlatONnetwork/PlatON-Go/common/vm"
	"github.com/PlatONnetwork/PlatON-Go/rlp"

	"github.com/PlatONnetwork/PlatON-Go/crypto/bls"
	"github.com/PlatONnetwork/PlatON-Go/ethclient"
	"github.com/PlatONnetwork/PlatON-Go/p2p/discover"
)

type stakingCases struct {
	commonCases
	paramsPath string
	params     *stakingParams
}

type stakingParams struct {
	Nodes []*nodeMem `json:"nodes"`
}

type nodeMem struct {
	Id     string `json:"id"`
	BlsKey string `json:"blsKey"`
	Url    string `json:"url"`
}

func (s *stakingCases) Prepare() error {
	if err := s.commonCases.Prepare(); err != nil {
		return err
	}
	s.paramsPath = path.Join(config.Dir, config.StakingConfigFile)
	s.parseConfigJson()
	return nil
}

func (s *stakingCases) parseConfigJson() {
	bytes, err := ioutil.ReadFile(s.paramsPath)
	if err != nil {
		panic(fmt.Errorf("parse stakingCases config file error,%s", err.Error()))
	}
	if err := json.Unmarshal(bytes, &s.params); err != nil {
		panic(fmt.Errorf("parse stakingCases config to json error,%s", err.Error()))
	}
	fmt.Print()
}

func (s *stakingCases) Start() error {

	return nil
}

func (s *stakingCases) Exec(caseName string) error {
	return s.commonCases.exec(caseName, s)
}

func (s *stakingCases) End() error {
	if err := s.commonCases.End(); err != nil {
		return err
	}
	if len(s.errors) != 0 {
		for _, value := range s.errors {
			s.SendError("restricting", value)
		}
		return errors.New("run restrictCases fail")
	}
	return nil
}

func (s *stakingCases) List() []string {
	return s.list(s)
}

func (s *stakingCases) CaseBatchDelegate() error {

	accountMap := make(map[string]*PriAccount, 0)
	accountQueue := make([]*PriAccount, 0)
	// loading all account

	index := 0
	for {

		obj := AccountPool.Get()

		if nil == obj {
			break
		}

		account, _ := obj.(*PriAccount)
		accountQueue = append(accountQueue, account)
		accountMap[account.Address.Hex()] = account
		index++

	}

	// 等待交易被确认
	waitFn := func(client *ethclient.Client, ctx context.Context, txHash common.Hash) error {
		i := 0
		for {
			if i > 10 {
				return errors.New("wait to long")
			}
			_, isPending, err := client.TransactionByHash(ctx, txHash)
			if err == nil {
				if !isPending {
					break
				}
			} else {
				if err != platON.NotFound {
					return err
				}
			}
			time.Sleep(time.Second)
			i++
		}
		return nil
	}

	// 查询结果的函数
	getresFn := func(client *ethclient.Client, ctx context.Context, txHash common.Hash) (*xcom.Result, error) {
		receipt, err := client.TransactionReceipt(ctx, txHash)
		if err != nil {
			return nil, fmt.Errorf("get TransactionReceipt fail:%v", err)
		}
		var logRes [][]byte
		if err := rlp.DecodeBytes(receipt.Logs[0].Data, &logRes); err != nil {
			return nil, fmt.Errorf("rlp decode fail:%v", err)
		}
		var res xcom.Result
		if err := json.Unmarshal(logRes[0], &res); err != nil {
			return nil, fmt.Errorf("json decode fail:%v", err)
		}
		return &res, nil
	}

	type socket struct {
		client *ethclient.Client
		ctx    context.Context
		nodeId discover.NodeID
	}

	clientQueue := make([]*socket, len(s.params.Nodes))

	var wg sync.WaitGroup

	type txResult struct {
		msg  string
		err  error
		hash common.Hash
	}

	stakeResCh := make(chan *txResult, len(s.params.Nodes))

	// 先质押一波节点
	for i := 0; i < len(s.params.Nodes); i++ {

		node := (s.params.Nodes)[i]
		nodeId := discover.MustHexID(node.Id)

		ctx := context.Background()
		// 创建需要质押的节点的链接
		client, err := ethclient.Dial(node.Url)
		if err != nil {
			return err
		}

		//s.client = client

		sock := &socket{
			client: client,
			ctx:    ctx,
			nodeId: nodeId,
		}

		clientQueue[i] = sock

		versionValue, err := client.GetProgramVersion(ctx)

		if err != nil {
			return err
		}

		proof, err := client.GetSchnorrNIZKProve(ctx)

		if err != nil {
			return err
		}

		// programVersionSign
		var versionSign common.VersionSign
		versionSign.UnmarshalText([]byte(versionValue.Sign))

		// bls publicKey
		var keyHex bls.PublicKeyHex
		keyHex.UnmarshalText([]byte(node.BlsKey))
		// bls proof
		var proofHex bls.SchnorrProofHex
		proofHex.UnmarshalText([]byte(proof))

		stakeAmount, _ := new(big.Int).SetString("10000000000000000000000000", 10)

		index := rand.Intn(len(accountQueue) - 1)

		stakingAccount := accountQueue[index]

		var params [][]byte
		params = make([][]byte, 0)
		fnType, _ := rlp.EncodeToBytes(uint16(1000))
		typ, _ := rlp.EncodeToBytes(uint16(0))
		benefitAddress, _ := rlp.EncodeToBytes(stakingAccount.Address)
		nodeIdrlp, _ := rlp.EncodeToBytes(nodeId)
		externalId, _ := rlp.EncodeToBytes("")
		nodeName, _ := rlp.EncodeToBytes("非官方第" + fmt.Sprint(i+1) + "号节点")
		website, _ := rlp.EncodeToBytes("")
		details, _ := rlp.EncodeToBytes("")
		amount, _ := rlp.EncodeToBytes(stakeAmount)
		programVersion, _ := rlp.EncodeToBytes(versionValue.Version)
		programVersionSign, _ := rlp.EncodeToBytes(versionSign)
		blsPubKey, _ := rlp.EncodeToBytes(keyHex)
		proofRlp, _ := rlp.EncodeToBytes(proofHex)

		params = append(params, fnType)
		params = append(params, typ)
		params = append(params, benefitAddress)
		params = append(params, nodeIdrlp)
		params = append(params, externalId)
		params = append(params, nodeName)
		params = append(params, website)
		params = append(params, details)
		params = append(params, amount)
		params = append(params, programVersion)
		params = append(params, programVersionSign)
		params = append(params, blsPubKey)
		params = append(params, proofRlp)

		send := s.encodePPOS(params)

		//fmt.Println("Staking from", stakingAccount.Address.String())

		txHash, err := SendRawTransaction(ctx, client, stakingAccount, vm.StakingContractAddr, "0", send)
		if err != nil {
			return fmt.Errorf("createStakingTransaction fail:%v", err)
		}
		log.Printf("Staking txHash %s \n", txHash.Hex())

		wg.Add(1)

		go func(txhash common.Hash, client *ethclient.Client, ctx context.Context) {

			res := new(txResult)

			if err := waitFn(client, ctx, txhash); err != nil {
				res.msg = "Failed to wait staking tx"
				res.err = err
				res.hash = txhash
				stakeResCh <- res
				wg.Done()
				return
			}

			xres, err := getresFn(client, ctx, txhash)
			if err != nil {
				res.msg = "Failed to query staking result"
				res.err = err
				res.hash = txhash
				stakeResCh <- res
				wg.Done()
				return
			}
			log.Printf("end create staking,res:%+v", xres)

			res.msg = "Staking success, " + fmt.Sprintf("res: %+v", xres)
			res.hash = txhash
			stakeResCh <- res
			wg.Done()

		}(txHash, client, ctx)
	}

	wg.Wait()
	close(stakeResCh)

	// 打印质押结果
	for res := range stakeResCh {

		if nil != res.err {
			log.Println("txHash:"+res.hash.Hex(), res.msg, "err:"+res.err.Error())
		} else {
			log.Println("txHash:"+res.hash.Hex(), res.msg)
		}
	}

	log.Printf("socket queue: %+v", clientQueue)

	minimumThreshold, _ := new(big.Int).SetString("10000000000000000000", 10)
	_, to := s.generateEmptyAccount()

	// 批量发起委托
	for len(accountMap) != 0 {

		i := rand.Intn(len(accountQueue))
		from := accountQueue[i]

		// 发起委托
		in := rand.Intn(len(clientQueue))
		sock := clientQueue[in]

		balance := s.GetBalance(sock.ctx, from.Address, nil)

		double := new(big.Int).Mul(minimumThreshold, big.NewInt(2))

		if balance.Cmp(common.Big0) == 0 || balance.Cmp(double) < 0 {

			accountQueue = append(accountQueue[:i], accountQueue[i:]...)
			delete(accountMap, from.Address.Hex())
			AccountPool.Put(from)
			continue
		} else {

			// 委托
			fnType, _ := rlp.EncodeToBytes(uint16(1004))
			encodeTyp, _ := rlp.EncodeToBytes(uint16(0))
			id, _ := rlp.EncodeToBytes(sock.nodeId)
			encodeAmount, _ := rlp.EncodeToBytes(minimumThreshold)

			var params [][]byte
			params = append(params, fnType)
			params = append(params, encodeTyp)
			params = append(params, id)
			params = append(params, encodeAmount)
			send := s.encodePPOS(params)

			txhash, err := SendRawTransaction(sock.ctx, sock.client, from, vm.StakingContractAddr, "0", send)

			if nil != err {
				log.Printf("Failed to delegate tx: %s, err: %v", txhash.Hex(), err)
				continue
			}

			// 转账
			txHash2, err := SendRawTransaction(sock.ctx, sock.client, from, to, minimumThreshold.String(), nil)
			if err != nil {
				log.Printf("Failed to transfer tx: %s, err: %v", txHash2.Hex(), err)
				continue
			}

			wg.Add(1)

			go func(client *ethclient.Client, ctx context.Context, delTxHash, txsorHash common.Hash) {

				// 等 delegate 交易确认
				if err := waitFn(client, ctx, delTxHash); err != nil {
					log.Printf("Failed to wait the delegate tx: %s, err: %v", delTxHash.Hex(), err)
					wg.Done()
					return
				}

				xres, err := getresFn(client, ctx, delTxHash)
				if err != nil {
					log.Printf("Failed to query delegate result tx: %s, err: %v", delTxHash, err)
					wg.Done()
					return
				}

				log.Printf("Delegate success tx: %s, res: %+v", delTxHash.Hex(), xres)

				// 等 转账交易确认
				if err := waitFn(client, ctx, txsorHash); err != nil {
					log.Printf("Failed to wait the transfer tx: %s, err: %v", txsorHash.Hex(), err)
					wg.Done()
					return
				}

				log.Printf("transfer success tx: %s", txsorHash.Hex())
				wg.Done()

			}(sock.client, sock.ctx, txhash, txHash2)

		}
	}

	wg.Wait()
	log.Println("Batch Delegate finished")
	return nil

}
