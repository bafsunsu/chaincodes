package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/hyperledger/fabric-contract-api-go/contractapi"
)

// ProductContract provides functions for managing an Asset
type ProductContract struct {
	contractapi.Contract
}

type Product struct {
	ID       string `json:"ID"`
	Name     string `json:"Name_Test"`
	Quantity int    `json:"Quantity"`
	Owner    string `json:"Owner"`
}

//Custom InitLedger  to Initiate the ledger with intial values if needed
func (p *ProductContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	products := []Product{
		{ID: "1", Name: "Wheat", Quantity: 100, Owner: "Sunsu"},
		{ID: "2", Name: "Rice", Quantity: 100, Owner: "Sunsu"},
		{ID: "3", Name: "Oats", Quantity: 100, Owner: "Sunsu"},
	}
	for _, asset := range products {
		assetJSON, err := json.Marshal(asset)
		if err != nil {
			return err
		}

		err = ctx.GetStub().PutState(asset.ID, assetJSON)
		if err != nil {
			return fmt.Errorf("failed to put to world state. %v", err)
		}
	}

	return nil
}

//AssetExists Generic method to verify key already is recorded or not
func (s *ProductContract) AssetExists(ctx contractapi.TransactionContextInterface, id string) (bool, error) {
	assetJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return false, fmt.Errorf("failed to read from world state: %v", err)
	}

	return assetJSON != nil, nil
}

func (p *ProductContract) CreateProduct(ctx contractapi.TransactionContextInterface, id string, name string, quantity int, owner string) error {

	exists, err := p.AssetExists(ctx, id)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("the asset %s already exists", id)
	}

	product := Product{
		ID:       id,
		Name:     name,
		Quantity: quantity,
		Owner:    owner,
	}
	productJSON, err := json.Marshal(product)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, productJSON)

}

func (p ProductContract) ReadProduct(ctx contractapi.TransactionContextInterface, id string) (*Product, error) {
	productJSON, err := ctx.GetStub().GetState(id)
	if err != nil {
		return nil, fmt.Errorf("failed to read from world state: %v", err)
	}
	if productJSON == nil {
		return nil, fmt.Errorf("the asset %s does not exist", id)
	}

	var product Product
	err = json.Unmarshal(productJSON, &product)
	if err != nil {
		return nil, err
	}

	return &product, nil
}

func (p ProductContract) ChangeOwner(ctx contractapi.TransactionContextInterface, id string, newOwner string) error {
	product, err := p.ReadProduct(ctx, id)
	if err != nil {
		return err
	}

	product.Owner = newOwner
	assetJSON, err := json.Marshal(product)
	if err != nil {
		return err
	}

	return ctx.GetStub().PutState(id, assetJSON)
}

func (p ProductContract) DeleteProduct(ctx contractapi.TransactionContextInterface, id string) error {
	_, err := p.ReadProduct(ctx, id)
	if err != nil {
		return err
	}

	return ctx.GetStub().DelState(id)
}

// GetAllAssets returns all assets found in world state
func (s *ProductContract) GetAllAssets(ctx contractapi.TransactionContextInterface) ([]*Product, error) {
	// range query with empty string for startKey and endKey does an
	// open-ended query of all assets in the chaincode namespace.
	resultsIterator, err := ctx.GetStub().GetStateByRange("", "")
	if err != nil {
		return nil, err
	}
	defer resultsIterator.Close()

	var assets []*Product
	for resultsIterator.HasNext() {
		queryResponse, err := resultsIterator.Next()
		if err != nil {
			return nil, err
		}

		var asset Product
		err = json.Unmarshal(queryResponse.Value, &asset)
		if err != nil {
			return nil, err
		}
		assets = append(assets, &asset)
	}

	return assets, nil
}

func main() {
	productChaincode, err := contractapi.NewChaincode(&ProductContract{})
	if err != nil {
		log.Panicf("Error creating asset-transfer-basic chaincode: %v", err)
	}

	if err := productChaincode.Start(); err != nil {
		log.Panicf("Error starting asset-transfer-basic chaincode: %v", err)
	}
}

func (t *ProductContract) GetHistoryForRecord(ctx contractapi.TransactionContextInterface, id string) (string, error) {

	recordKey := id

	fmt.Printf("- start getHistoryForRecord: %s\n", recordKey)

	resultsIterator, err := ctx.GetStub().GetHistoryForKey(recordKey)
	if err != nil {
		return "", err
	}
	defer resultsIterator.Close()

	// buffer is a JSON array containing historic values for the key/value pair
	var buffer bytes.Buffer
	buffer.WriteString("[")

	bArrayMemberAlreadyWritten := false
	for resultsIterator.HasNext() {
		response, err := resultsIterator.Next()
		if err != nil {
			return "", err
		}
		// Add a comma before array members, suppress it for the first array member
		if bArrayMemberAlreadyWritten == true {
			buffer.WriteString(",")
		}
		buffer.WriteString("{\"TxId\":")
		buffer.WriteString("\"")
		buffer.WriteString(response.TxId)
		buffer.WriteString("\"")

		buffer.WriteString(", \"Value\":")
		// if it was a delete operation on given key, then we need to set the
		//corresponding value null. Else, we will write the response.Value
		//as-is (as the Value itself a JSON vehiclePart)
		if response.IsDelete {
			buffer.WriteString("null")
		} else {
			buffer.WriteString(string(response.Value))
		}

		buffer.WriteString(", \"Timestamp\":")
		buffer.WriteString("\"")
		buffer.WriteString(time.Unix(response.Timestamp.Seconds, int64(response.Timestamp.Nanos)).String())
		buffer.WriteString("\"")

		buffer.WriteString(", \"IsDelete\":")
		buffer.WriteString("\"")
		buffer.WriteString(strconv.FormatBool(response.IsDelete))
		buffer.WriteString("\"")

		buffer.WriteString("}")
		bArrayMemberAlreadyWritten = true
	}
	buffer.WriteString("]")

	fmt.Printf("- getHistoryForRecord returning:\n%s\n", buffer.String())

	return string(buffer.Bytes()), nil
}
