package wallet

import (
	"fmt"
	"github.com/ribencong/go-youPipe/service"
	"golang.org/x/crypto/ed25519"
	"sync"
	"time"
)

const SysTimeFormat = "2006-01-02 15:04:05"
const PipeDialTimeOut = time.Second * 2

func (w *Wallet) Running(done chan error) {

	done <- fmt.Errorf("payment channel exit")

	for {
		bill := &service.PipeBill{}
		if err := w.payConn.ReadJsonMsg(bill); err != nil {
			fmt.Printf("payment channel closed: %v", err)
			return
		}

		fmt.Printf("(%s)Got new bill:%s from:%s", time.Now().Format(SysTimeFormat), bill.String(), w.minerID)

		if err := w.counter.checkBill(bill, w.minerID); err != nil {
			fmt.Println(err)
			return
		}
		proof, err := w.counter.signBill(bill, w.acc.Key.PriKey)
		if err != nil {
			fmt.Print(err)
			return
		}

		if err := w.payConn.WriteJsonMsg(proof); err != nil {
			fmt.Printf("\nwrite back bill msg err:%v", err)
			return
		}

		if err := w.counter.updateLocalCounter(bill.UsedBandWidth); err != nil {
			fmt.Println(err.Error())
			return
		}
	}
}

type FlowCounter struct {
	sync.RWMutex
	closed    bool
	totalUsed int64
	unSigned  int64
	token     int64
}

func (p *FlowCounter) ToString() string {
	return fmt.Sprintf("close:%t totalUsed:%d unsigned:%d", p.closed, p.totalUsed, p.unSigned)
}

func (p *FlowCounter) isClosed() bool {
	p.RLock()
	defer p.RUnlock()
	return p.closed
}

func (p *FlowCounter) finish() {
	p.Lock()
	defer p.Unlock()
	p.closed = true
}

func (p *FlowCounter) updateLocalCounter(usedBand int64) error {
	p.Lock()
	defer p.Unlock()

	p.unSigned -= usedBand
	p.totalUsed += usedBand
	fmt.Printf("\nafter update:sign bill unSigned:%d total:%d\n", p.unSigned, p.totalUsed)

	if p.token+p.unSigned < 0 {
		return fmt.Errorf("\n charge bill out of control (%d)\n", p.unSigned)
	}

	return nil
}

func (p *FlowCounter) checkBill(bill *service.PipeBill, minerPubKey ed25519.PublicKey) error {

	pubKey := make(ed25519.PublicKey, ed25519.PublicKeySize)
	copy(pubKey, minerPubKey)

	if ok := bill.Verify(minerPubKey); !ok {
		return fmt.Errorf("miner's signature failed")
	}

	p.RLock()
	defer p.RUnlock()
	fmt.Printf("\nbefore update:sign  bill unSigned:%d total:%d\n", p.unSigned, p.totalUsed)

	if bill.UsedBandWidth > p.unSigned {
		fmt.Printf("\nI don't use so much bandwith user:(%d) unsigned(%d)\n", bill.UsedBandWidth, p.unSigned)
	}
	return nil
}

func (p *FlowCounter) signBill(bill *service.PipeBill, priKey ed25519.PrivateKey) (*service.PipeProof, error) {

	proof := &service.PipeProof{
		PipeBill: bill,
	}

	if err := proof.Sign(priKey); err != nil {
		fmt.Printf("Sign bill err:%v", err)
		return nil, err
	}

	return proof, nil
}

func (p *FlowCounter) Consume(n int) {

	//fmt.Printf("\t*******used:unSigned:%d, consume:%d\n", p.unSigned, n)

	p.Lock()
	defer p.Unlock()

	p.unSigned += int64(n)
}
