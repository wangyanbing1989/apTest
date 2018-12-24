package apTest

import (
	"fmt"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	"testing"
)

func mockInit(t *testing.T, stub *shim.MockStub, args [][]byte) {
	res := stub.MockInit("1", args)
	if res.Status != shim.OK {
		fmt.Println("Init failed", string(res.Message))
		t.FailNow()
	}
}

func TestAmicoPayChaincode_Init(t *testing.T) {
	//模拟链码部署
	scc := new(AmicoPayChaincode)
	stub := shim.NewMockStub("StudentChaincode", scc)
	initArgs :=[][]byte{[]byte("0.1"),[]byte("admin"),[]byte("admin")}
	mockInit(t, stub, initArgs)
}

func TestAmicoPayChaincode_Invoke(t *testing.T) {
	scc := new(AmicoPayChaincode)
	stub := shim.NewMockStub("StudentChaincode", scc)
	//createUserArgs :=[][]byte{[]byte("createUser"),[]byte("test21212"),[]byte("test")}
	createTransferArgs := [][]byte{[]byte("createTransfer"), []byte("amico"), []byte("admin"), []byte("test"), []byte("0.001"), []byte("test"), []byte("[]"), []byte("")}
	fmt.Println(stub.MockInvoke("1",createTransferArgs))
}



func Test(t *testing.T) {
	scc := new(AmicoPayChaincode)
	stub := shim.NewMockStub("StudentChaincode", scc)
	initArgs :=[][]byte{[]byte("0.1"),[]byte("admin"),[]byte("admin")}
	mockInit(t, stub, initArgs)

	createUserArgs :=[][]byte{[]byte("createUser"),[]byte("test"),[]byte("test")}
	fmt.Println(stub.MockInvoke("1",createUserArgs))

	createUserArgs1 :=[][]byte{[]byte("createUser"),[]byte("test1"),[]byte("test")}
	fmt.Println(stub.MockInvoke("1",createUserArgs1))

	rechargeArgs := [][]byte{[]byte("rechargeInterface"),[]byte("admin"),[]byte("test1"),[]byte("amico"),[]byte("100.0")}
	fmt.Println(stub.MockInvoke("1",rechargeArgs))

	createTransferArgs := [][]byte{[]byte("createTransactionInterface"), []byte("amico"), []byte("test1"), []byte("test"), []byte("90"), []byte("test"), []byte("[]"), []byte("")}
	fmt.Println(stub.MockInvoke("1",createTransferArgs))

	withdrawArgs := [][]byte{[]byte("withdrawInterface"),[]byte("admin"),[]byte("test"),[]byte("test"),[]byte("amico"),[]byte("81")}
	fmt.Println(stub.MockInvoke("1",withdrawArgs))

	transactionInfoArgs := [][]byte{[]byte("transactionInfo"),[]byte("admin"),[]byte("3"),[]byte("amico")}
	fmt.Println(stub.MockInvoke("1",transactionInfoArgs))
}

