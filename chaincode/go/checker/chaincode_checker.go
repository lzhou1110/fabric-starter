
package main

import (
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
	"encoding/pem"
	"crypto/x509"
	"strings"
)

var logger = shim.NewLogger("CheckerChaincode")

type Credit struct {
	Borrower 	string		`json:"borrower"`
	Amount 		int			`json:"amount"`
}

type CheckerChaincode struct {
}

func (t *CheckerChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Debug("Init")
	return shim.Success(nil)
}

func (t *CheckerChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {

	logger.Debug("Invoke")

	function, args := stub.GetFunctionAndParameters()
	if function == "bankrupt" {
		return t.bankrupt(stub, args)
	} else if function == "tolerate" {
		return t.tolerate(stub, args)
	}
	return pb.Response{Status:403, Message:"Invalid invoke function name."}
}

func (t *CheckerChaincode) bankrupt(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	// Checking organization access right
	creatorBytes, err := stub.GetCreator()
	if err != nil {
		return shim.Error(err.Error())
	}
	_, org := getCreator(creatorBytes)
	if org != "system" {
		return pb.Response{Status:401,Message:"cannot call method"}
	}

	// Reading input
	if len(args) != 1 {
		return pb.Response{Status:403,Message:"Incorrect number of arguments"}
	}
	currentDate := args[0]
	logger.Debugf("currentDate=%s", currentDate)

	functionArgs := make([][]byte, 2)
	functionArgs[0] = []byte("due")
	functionArgs[1] = []byte(currentDate)

	response := stub.InvokeChaincode("loan", functionArgs, "relationship")

	logger.Debugf("payload=%s", string(response.Payload))

	return shim.Success(nil)
}

func (t *CheckerChaincode) tolerate(stub shim.ChaincodeStubInterface, args []string) pb.Response {
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
	err := shim.Start(new(CheckerChaincode))
	if err != nil {
		logger.Error(err.Error())
	}
}
