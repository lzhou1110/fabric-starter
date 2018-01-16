
package main

import (
	"strconv"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"encoding/pem"
	"crypto/x509"
	"strings"
	"time"
	"encoding/json"
)

var logger = shim.NewLogger("LoanChaincode")

type LoanValue struct {
	Amount     int            `json:"amount"`
	Due        string         `json:"due"`
}

// LoanChaincode example simple Chaincode implementation
type LoanChaincode struct {
}

func (t *LoanChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Init")

	return shim.Success(nil)
}

func (t *LoanChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Invoke")

	function, args := stub.GetFunctionAndParameters()
	if function == "lend" {
		return t.lend(stub, args)
	} else if function == "pay" {
		return t.pay(stub, args)
	} else if function == "due" {
		return t.due(stub, args)
	}

	return pb.Response{Status:403, Message:"Invalid invoke function name."}
}

func (t *LoanChaincode) lend(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 4 {
		return pb.Response{Status:403,Message:"Incorrect number of arguments"}
	}

	borrower := args[0]

	amount := args[1]
	amountVal, err := strconv.Atoi(amount)
	if err != nil {
		return pb.Response{Status:403,Message:"Cannot convert to int"}
	}

	due := args[2]
	logger.Debugf("due=%s", due)

	dueDate, err := time.Parse("2006-01-02", due)
	if err != nil {
		logger.Error(err)
		return pb.Response{Status:403,Message:"Cannot convert to Time"}
	}

	tolerance := args[3]
	toleranceVal, err := strconv.Atoi(tolerance)
	if err != nil {
		return pb.Response{Status:403,Message:"Cannot convert to int"}
	}

	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}

	lender, org := getCreator(creatorBytes)

	if org != "lender" {
		//return pb.Response{Status:401,Message:"Cannot call method"}
	}

	logger.Debugf("lender=%s borrower=%s amount=%d due=%s tolerance=%d", lender, borrower, amountVal,
		dueDate.String(), toleranceVal)

	ck, _ := stub.CreateCompositeKey("Loan", []string{borrower, lender})

	loanValue := LoanValue{Amount:amountVal, Due:due}

	bytesLoanValue, err := json.Marshal(loanValue)

	err = stub.PutState(ck, bytesLoanValue)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(bytesLoanValue)
}

func (t *LoanChaincode) pay(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return pb.Response{Status:403,Message:"Incorrect number of arguments"}
	}

	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}

	borrower, org := getCreator(creatorBytes)

	if org != "borrower" {
		//return pb.Response{Status:401,Message:"Cannot call method"}
	}

	lender := args[0]

	amount := args[1]
	amountVal, err := strconv.Atoi(amount)
	if err != nil {
		return pb.Response{Status:403,Message:"Cannot convert to int"}
	}

	ck, _ := stub.CreateCompositeKey("Loan", []string{borrower, lender})

	loanBytes, err := stub.GetState(ck)
	if err != nil {
		return shim.Error(err.Error())
	}

	if loanBytes == nil {
		return pb.Response{Status:404,Message:"Loan not found"}
	}

	var loanValue LoanValue
	err = json.Unmarshal(loanBytes, &loanValue)
	if err != nil {
		return shim.Error(err.Error())
	}

	loanValue.Amount = loanValue.Amount - amountVal

	loanBytes, err = json.Marshal(loanValue)
	if err != nil {
		return shim.Error(err.Error())
	}

	stub.PutState(ck, loanBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(loanBytes)
}

func (t *LoanChaincode) due(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 1 {
		return pb.Response{Status:403,Message:"Incorrect number of arguments"}
	}

	it, err := stub.GetStateByPartialCompositeKey("Loan", []string{})

	if err != nil {
		return shim.Error(err.Error())
	}
	defer it.Close()

	for it.HasNext() {
		next, err := it.Next()
		if err != nil {
			return shim.Error(err.Error())
		}

		var loanValue LoanValue
		err = json.Unmarshal(next.Value, &loanValue)
		if err != nil {
			return shim.Error(err.Error())
		}

		_, keys, err := stub.SplitCompositeKey(next.Key)
		if err != nil {
			return shim.Error(err.Error())
		}

		borrower := keys[0]
		lender := keys[1]
		amount := loanValue.Amount
		due := loanValue.Due

		logger.Debugf("borrower=%s lender=%s amount=%d due=%s", borrower, lender, amount, due)
	}

	return shim.Success(nil)
}

var getCreator = func (certificate []byte) (string, string) {
	data := certificate[strings.Index(string(certificate), "-----"): strings.LastIndex(string(certificate), "-----")+5]
	block, _ := pem.Decode([]byte(data))
	cert, _ := x509.ParseCertificate(block.Bytes)
	organization := cert.Issuer.Organization[0]
	commonName := cert.Subject.CommonName
	logger.Debug("commonName: " + commonName + ", organization: " + organization)

	organizationShort := strings.Split(organization, ".")[0]

	return commonName, organizationShort
}

func main() {
	err := shim.Start(new(LoanChaincode))
	if err != nil {
		logger.Error(err.Error())
	}
}
