package wallet

import (
	"fmt"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"golang.org/x/crypto/ed25519"
	"time"
)

const SysTimeFormat = "2006-01-02 15:04:05"
const PipeDialTimeOut = time.Second * 2

func (w *Wallet) Running() {

	defer w.payConn.Close()
	defer func() {
		w.FlowCounter.Closed = true
	}()

	for {
		bill := &service.PipeBill{}
		if err := w.payConn.ReadJsonMsg(bill); err != nil {
			w.fatalErr <- fmt.Errorf("payment channel Closed: %v", err)
			return
		}

		fmt.Printf("(%s)Got new bill:%s",
			time.Now().Format(SysTimeFormat), bill.String())

		pubKey := account.ID(w.minerID)
		fmt.Printf("\nPipeBill Wallet socks ID:%s key:%s", w.minerID, pubKey)
		proof, err := w.signBill(bill, pubKey, w.Key.PriKey)
		if err != nil {
			w.fatalErr <- err
			return
		}

		if err := w.payConn.WriteJsonMsg(proof); err != nil {
			w.fatalErr <- err
			return
		}
	}
}

func (p *FlowCounter) signBill(bill *service.PipeBill, minerId account.ID, priKey ed25519.PrivateKey) (*service.PipeProof, error) {

	if ok := bill.Verify(minerId); !ok {
		return nil, fmt.Errorf("miner's signature failed")
	}

	p.Lock()
	defer p.Unlock()

	if bill.UsedBandWidth > p.unSigned {
		return nil, fmt.Errorf("\n\nI don't use so much bandwith user:(%d) unsigned(%d)", bill.UsedBandWidth, p.unSigned)
	}

	proof := &service.PipeProof{
		PipeBill: bill,
	}

	if err := proof.Sign(priKey); err != nil {
		return nil, err
	}

	fmt.Printf("\n\n sign  bill unSigned:%d total:%d", p.unSigned, p.totalUsed)
	p.unSigned -= bill.UsedBandWidth
	p.totalUsed += bill.UsedBandWidth

	return proof, nil
}

func (p *FlowCounter) Consume(n int) {

	//fmt.Printf("\t*******used:unSigned:%d, consume:%d\n", p.unSigned, n)

	p.Lock()
	defer p.Unlock()

	p.unSigned += int64(n)
}
