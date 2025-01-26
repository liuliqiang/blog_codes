package main

import (
	"log"

	"github.com/dtm-labs/client/dtmcli"
	"github.com/gin-gonic/gin"
	"github.com/lithammer/shortuuid/v3"
)

const (
	dtmServer   = "http://localhost:36789/api/dtmsvr"
	fromBankURL = "http://localhost:9010/api/citi_bank"
	toBankURL   = "http://localhost:9020/api/hsbc_bank"
)

// QsFireRequest quick start: fire request
func main() {
	req := &gin.H{"amount": 30} // the payload of requests
	gid := shortuuid.New()
	// DtmServer is the address of dtm
	saga := dtmcli.NewSaga(dtmServer, gid).
		// add a branch transaction，action url is: qsBusi+"/TransOut"， compensate url: qsBusi+"/TransOutCompensate"
		Add(fromBankURL+"/TransOut", fromBankURL+"/TransOutCompensate", req).
		// add a branch transaction，action url is: qsBusi+"/TransIn"， compensate url: qsBusi+"/TransInCompensate"
		Add(toBankURL+"/TransIn", toBankURL+"/TransInCompensate", req)
	// submit saga global transaction，dtm will finish all action and compensation
	err := saga.Submit()

	if err != nil {
		log.Printf("transaction: %s failed: %v", saga.Gid, err)
	} else {
		log.Printf("transaction: %s success", saga.Gid)
	}
}
