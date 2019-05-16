package wallet

import (
	"fmt"
	"github.com/ribencong/go-youPipe/account"
	"github.com/ribencong/go-youPipe/service"
	"github.com/ribencong/go-youPipe/utils"
	"golang.org/x/crypto/ed25519"
	"time"
)

func (w *Wallet) payMonitor() {
	for {
		bill := &service.PipeBill{}
		if err := w.payConn.ReadJsonMsg(bill); err != nil {
			w.fatalErr <- fmt.Errorf("payment channel Closed: %v", err)
			return
		}

		fmt.Printf("(%s)Got new bill:%s",
			time.Now().Format(utils.SysTimeFormat), bill.String())

		proof, err := w.signBill(bill, w.curService.ID, w.Key.PriKey)
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

	fmt.Printf("\t*******used:unSigned:%d, consume:%d\n", p.unSigned, n)

	p.Lock()
	defer p.Unlock()

	p.unSigned += int64(n)
}