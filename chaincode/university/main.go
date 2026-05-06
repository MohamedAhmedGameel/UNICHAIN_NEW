package main

import (
	"log"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

func main() {
	cc, err := contractapi.NewChaincode(&UniversityChaincode{})
	if err != nil {
		log.Fatalf("Error creating university chaincode: %v", err)
	}
	if err := cc.Start(); err != nil {
		log.Fatalf("Error starting university chaincode: %v", err)
	}
}
