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
		w.counter.Closed = true
	}()

	for {
		bill := &service.PipeBill{}
		if err := w.payConn.ReadJsonMsg(bill); err != nil {
			w.fatalErr <- fmt.Errorf("payment channel Closed: %v", err)
			return
		}

		fmt.Printf("(%s)Got new bill:%s", time.Now().Format(SysTimeFormat), bill.String())

		fmt.Printf("\n---1--->=:PipeBill Wallet socks ID:%s wallet:%s", w.minerID, w.ToString())
		proof, err := w.counter.signBill(bill, account.ID(w.minerID), w.acc.Key.PriKey)
		if err != nil {
			fmt.Print(err)
			w.fatalErr <- err
			return
		}
		fmt.Printf("\n---2--->=:PipeBill Wallet socks ID:%s wallet:%s", w.minerID, w.ToString())
		if err := w.payConn.WriteJsonMsg(proof); err != nil {
			fmt.Print(err)
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
		fmt.Println(err)
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
