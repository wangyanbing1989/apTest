package apTest
//noinspection GoUnsortedImport
import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"crypto/md5"
	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

type AmicoPayChaincode struct {
}

type Paras struct {
	TransactionRate float64
	AdminUserId string
}

type TransactionId int64


type Transaction struct {
	TokenType string
	TransactionId int64
	FromUserId string
	ToUserId string
	Amount float64
	Frozen float64
	NeedApproval []string
	Approved []string
	Completed bool
	Remark string
	LastFromTransactionId int64
	LastToTransactionId int64
}

type User struct {
	UserId string
	Signature string
	RecivableBalance map[string]float64
	PayableBalance map[string]float64
	Balance map[string]float64
	LastTransactionIdList map[string]int64
	ThreadLock string
}

func (a *AmicoPayChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {

	// 从交易提案中获取参数
	args := stub.GetStringArgs()
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3(transactionRate, adminUserId, adminPwd)")
	}
	var err error
	var parasAsBytes []byte
	var transactionRate float64
	transactionRate, err = strconv.ParseFloat(args[0], 64)
	if err != nil {
		return shim.Error(err.Error())
	}
	adminUserId := args[1]
	adminPwd := args[2]
	adminCreate := []string{adminUserId,adminPwd}
	a.createUser(stub, adminCreate)

	paras := &Paras{transactionRate, adminUserId}
	//保存
	parasAsBytes, err = json.Marshal(paras)
	if err != nil{
		return shim.Error("marshal failed")
	}
	err = stub.PutState("Paras", parasAsBytes)
	if err != nil {
		return shim.Error("init para set failed")
	}

	//获取公共参数
	//pAsBytes, err := stub.GetState("Paras")
	//if err != nil {
	//	return shim.Error(err.Error())
	//}
	//var p Paras
	//err = json.Unmarshal(pAsBytes, &p)
	//if err != nil {
	//	return shim.Error(err.Error())
	//}

	return shim.Success(parasAsBytes)
}

func (a *AmicoPayChaincode) Invoke(stub shim.ChaincodeStubInterface) pb.Response {
	function, args := stub.GetFunctionAndParameters()
	fmt.Println("invoke is running " + function)
	if function == "createUser" {//自定义函数名称
		return a.createUser(stub, args)//定义调用的函数
	}else if function == "createTransactionInterface" {
		return a.createTransactionInterface(stub, args)
	}else if function == "confirm" {
		return a.confirm(stub, args)
	}else if function =="balance" {
		return a.balance(stub, args)
	}else if function =="transactionInfo" {
		return a.transactionInfo(stub, args)
	}else if function =="rechargeInterface" {
		return a.rechargeInterface(stub, args)
	}else if function =="withdrawInterface" {
		return a.withdrawInterface(stub, args)
	}
	return shim.Error("Received unknown function invocation")
}

//Interface





//Function
//获取新交易id（自增）
func (a *AmicoPayChaincode) transactionIdGet(stub shim.ChaincodeStubInterface) (id int64, err error) {
	var currentTransactionId int64
	currentTransactionIdAsBytes, err := stub.GetState("TransactionId")
	if err != nil {
		return 0, errors.New("transaction get error")
	}
	if currentTransactionIdAsBytes==nil {
		currentTransactionId = 0
	}else {
		err = json.Unmarshal(currentTransactionIdAsBytes, &currentTransactionId)
		if err != nil{
			return 0, errors.New("unmarshal failed")
		}
	}
	currentTransactionId = currentTransactionId+1
	//保存
	currentTransactionIdAsBytes, err = json.Marshal(currentTransactionId)
	if err != nil{
		return 0, errors.New("marshal failed")
	}
	err = stub.PutState("TransactionId", currentTransactionIdAsBytes)
	if err != nil {
		return 0, errors.New("transaction put error")
	}

	return currentTransactionId, nil
}

//创建用户
/*input:userid string
        password string
  output: success
          userexist
          improper-userid
          improper-pwd
          error
*/
func (a *AmicoPayChaincode) createUser(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	if len(args) != 2 {
		return shim.Error("Incorrect number of arguments. Expecting 2(userid, password)")
	}

	userId := args[0]
	signature := md5encode(args[1])
	user := &User{userId, signature, map[string]float64{},map[string]float64{},map[string]float64{}, map[string]int64{}, ""}

	//构建用户联合主键
	userKey, err := stub.CreateCompositeKey("User", []string{"user", userId})
	if err != nil {
		return shim.Error(err.Error())
	}

	//结构体转json
	userJSONasBytes, err := json.Marshal(user)
	if err != nil {
		return shim.Error(err.Error())
	}

	//保存
	err = stub.PutState(userKey, userJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(userJSONasBytes)

}

//转账交易
/*input:token string
        from-userid string
        to-userid string
        amount float32
        password string
        needApproval list
        remark string
  output: transactionId(if successed)
		  wrong-pwd
          unexist-from-userid
          unexist-to-userid
		  insufficient-balance
          improper-amount
		  imporper-approval
		  unexist-approval-userid
*/
func (a *AmicoPayChaincode) createTransactionInterface(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//获取入参
	if len(args) != 7 {
		return shim.Error("Incorrect number of arguments. Expecting 7(token, from, to, amount, password, needApproval, remark)")
	}

	token := args[0]
	from := args[1]
	to := args[2]
	amount, err := strconv.ParseFloat(args[3], 64)
	if err != nil {
		return shim.Error(err.Error())
	}
	password := args[4]
	needApproval := args[5]
	var needApprovalList []string
	var wo []string
	if err := json.Unmarshal([]byte(needApproval), &wo); err == nil {
		needApprovalList =wo
	}
	remark := args[6]

	transaction, err := a.createTransaction(stub,token, from, to, amount, password, needApprovalList, remark)
	if err != nil {
		return shim.Error(err.Error())
	}

	transactionJSONasBytes, err := json.Marshal(transaction)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(transactionJSONasBytes)
}



func (a *AmicoPayChaincode) confirm(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//获取数据
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3(userid, password, traditionid)")
	}
	userId := args[0]
	password := args[1]
	//构建用户联合主键
	userKey, err := stub.CreateCompositeKey("User", []string{"user", userId})
	if err != nil {
		return shim.Error(err.Error())
	}
	transactionId := args[2]

	//获取用户数据
	userAsBytes, err := stub.GetState(userKey)
	if err != nil {
		return shim.Error(err.Error())
	}else if userAsBytes == nil {
		return shim.Error("This user doesn't exists: " + userId)
	}
	var user User
	err = json.Unmarshal(userAsBytes, &user)
	if err != nil{
		return shim.Error("umarshall failed")
	}

	//验证密码是否正确
	signature := md5encode(password)
	if signature != user.Signature {
		return shim.Error("password incorrect")
	}

	//核准交易
	transactionAsBytes, err := stub.GetState(transactionId)
	if err != nil {
		return shim.Error(err.Error())
	}else if transactionAsBytes == nil {
		return shim.Error("This transaction doesn't exists")
	}

	var transaction Transaction
	err = json.Unmarshal(transactionAsBytes, transaction)
	if err != nil{
		return shim.Error("umarshall failed")
	}
	transaction.Approved = append(transaction.Approved, userKey)

	//结构体转json
	transactionJSONasBytes, err := json.Marshal(transaction)
	if err != nil {
		return shim.Error(err.Error())
	}

	//transaction提交
	err = stub.PutState(transactionId, transactionJSONasBytes)
	if err != nil {
		return shim.Error(err.Error())
	}

	//交易确认
	a.autoConfirm(stub, transactionId)

	return shim.Success(transactionJSONasBytes)

}

//信息查询
func (a *AmicoPayChaincode) balance(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	userId := args[0]
	//检查用户是否存在
	userKey, err := stub.CreateCompositeKey("User", []string{"user", userId})
	if err != nil {
		return shim.Error(err.Error())
	}
	userAsBytes, err := stub.GetState(userKey)
	if err != nil {
		return shim.Error(err.Error())
	}else if userAsBytes == nil {
		return shim.Error("This user doesn't exists: " + userId)
	}

	//获取余额
	var user User
	err = json.Unmarshal(userAsBytes, &user)
	if err != nil {
		return shim.Error(err.Error())
	}
	balance := user.Balance

	balanceAsBytes,err := json.Marshal(balance)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(balanceAsBytes)
}

func (a *AmicoPayChaincode) transactionInfo(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//获取数据
	if len(args) != 3 {
		return shim.Error("Incorrect number of arguments. Expecting 3(userid,lastTransactionId, token)")
	}
	userId := args[0]
	lastTransactionId := args[1]
	token := args[2]

	userKey, err := stub.CreateCompositeKey("User", []string{"user", userId})
	if err != nil {
		return shim.Error(err.Error())
	}

	var targetId int64

	if(lastTransactionId == "null") {
		userAsBytes, err := stub.GetState(userKey)
		if err != nil {
			return shim.Error(err.Error())
		}else if userAsBytes == nil {
			return shim.Error("This user doesn't exists: " + userId)
		}

		var user User
		err = json.Unmarshal(userAsBytes, &user)
		if err != nil {
			return shim.Error(err.Error())
		}
		targetId = user.LastTransactionIdList[token]

	}else {
		lastTransactionAsBytes, err := stub.GetState(lastTransactionId)
		if err != nil {
			return shim.Error(err.Error())
		}else if lastTransactionAsBytes == nil {
			return shim.Error("This transaction doesn't exists: " + lastTransactionId)
		}

		var lastTransaction Transaction
		err = json.Unmarshal(lastTransactionAsBytes, &lastTransaction)
		if err != nil {
			return shim.Error(err.Error())
		}

		if lastTransaction.FromUserId == userId {
			targetId = lastTransaction.LastFromTransactionId
		}else {
			targetId = lastTransaction.LastToTransactionId
		}
	}

	transactionAsBytes, err := stub.GetState(strconv.FormatInt(targetId,10))

	return shim.Success(transactionAsBytes)


}

//*******admin******//
//发币(仅支持对admin账户)
func (a *AmicoPayChaincode) offer(stub shim.ChaincodeStubInterface, adminPwd string, token string, amount float64) (num float64, e error) {
	//获取公共参数
	parasAsBytes, err := stub.GetState("Paras")
	if err != nil {
		return 0,errors.New("db get error")
	}
	var paras Paras
	err = json.Unmarshal(parasAsBytes, &paras)
	if err != nil {
		return 0,errors.New("unmarshal error")
	}
	adminUserId := paras.AdminUserId

	//获取admin
	//构建用户联合主键
	adminUserKey, err := stub.CreateCompositeKey("User", []string{"user", adminUserId})
	if err != nil {
		return 0,errors.New("userKey composite error")
	}
	adminUserAsBytes, err := stub.GetState(adminUserKey)
	if err != nil {
		return 0,errors.New("db get error")
	}else if adminUserAsBytes == nil {
		return 0,errors.New("This user doesn't exists: " + adminUserId)
	}
	var admin User
	err = json.Unmarshal(adminUserAsBytes, &admin)
	if err != nil{
		return 0,errors.New("umarshall failed")
	}

	//验证密码
	signature := md5encode(adminPwd)
	if signature != admin.Signature {
		return 0,errors.New("password incorrect")
	}

	//中心账户增发
	admin.Balance[token] = admin.Balance[token] + amount
	adminUserAsBytes, err = json.Marshal(admin)
	if err != nil{
		return 0,errors.New("marshall failed")
	}

	//数据上传
	err = stub.PutState(adminUserKey, adminUserAsBytes)
	if err != nil {
		return 0,errors.New("db put error")
	}

	return amount, nil
}

func (a *AmicoPayChaincode) burn(stub shim.ChaincodeStubInterface, adminPwd string, token string, amount float64) (num float64, e error) {
	//获取公共参数
	parasAsBytes, err := stub.GetState("Paras")
	if err != nil {
		return 0,errors.New("db get error")
	}
	var paras Paras
	err = json.Unmarshal(parasAsBytes, &paras)
	if err != nil {
		return 0,errors.New("unmarshal error")
	}
	adminUserId := paras.AdminUserId

	//获取admin
	//构建用户联合主键
	adminUserKey, err := stub.CreateCompositeKey("User", []string{"user", adminUserId})
	if err != nil {
		return 0,errors.New("userKey composite error")
	}
	adminUserAsBytes, err := stub.GetState(adminUserKey)
	if err != nil {
		return 0,errors.New("db get error")
	}else if adminUserAsBytes == nil {
		return 0,errors.New("This user doesn't exists: " + adminUserId)
	}
	var admin User
	err = json.Unmarshal(adminUserAsBytes, &admin)
	if err != nil{
		return 0,errors.New("umarshall failed")
	}

	//验证密码
	signature := md5encode(adminPwd)
	if signature != admin.Signature {
		return 0,errors.New("password incorrect")
	}

	//中心账户增发
	admin.Balance[token] = admin.Balance[token] - amount
	adminUserAsBytes, err = json.Marshal(admin)
	if err != nil{
		return 0,errors.New("marshall failed")
	}

	//数据上传
	err = stub.PutState(adminUserKey, adminUserAsBytes)
	if err != nil {
		return 0,errors.New("db put error")
	}

	return amount, nil
}

//充值
func (a *AmicoPayChaincode) rechargeInterface(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//获取数据
	if len(args) != 4 {
		return shim.Error("Incorrect number of arguments. Expecting 4(adminPwd, rechargeUserId, token, amount)")
	}

	adminPwd := args[0]
	rechargeUserId := args[1]
	token := args[2]
	amount, err := strconv.ParseFloat(args[3],64)
	if err != nil {
		return shim.Error("amount parse err")
	}
	needApproval :=[]string{}
	remark :="recharge"

	//获取公共参数
	parasAsBytes, err := stub.GetState("Paras")
	if err != nil {
		return shim.Error(err.Error())
	}
	var para Paras
	json.Unmarshal(parasAsBytes, &para)
	adminUserId := para.AdminUserId

	//增发
	_, err = a.offer(stub, adminPwd, token, amount)
	if err != nil {
		return shim.Error(err.Error())
	}

	//转账
	_, err = a.createTransaction(stub, token, adminUserId, rechargeUserId, amount, adminPwd, needApproval, remark)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("success"))
}

//提现
func (a *AmicoPayChaincode) withdrawInterface(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	//获取数据
	if len(args) != 5 {
		return shim.Error("Incorrect number of arguments. Expecting 4(adminPwd, withdrawUserId, withdrawUserPwd, token, amount)")
	}

	adminPwd := args[0]
	withdrawUserId := args[1]
	withdrawUserPwd := args[2]
	token := args[3]
	amount, err := strconv.ParseFloat(args[4],64)
	if err != nil {
		return shim.Error("amount parse err")
	}
	needApproval:= []string{}
	remark :="recharge"

	//获取公共参数
	parasAsBytes, err := stub.GetState("Paras")
	if err != nil {
		return shim.Error(err.Error())
	}
	var para Paras
	json.Unmarshal(parasAsBytes, &para)
	adminUserId := para.AdminUserId

	//转账
	_, err = a.createTransaction(stub, token, withdrawUserId, adminUserId, amount, withdrawUserPwd, needApproval, remark)
	if err != nil {
		return shim.Error(err.Error())
	}

	//销毁
	_, err = a.burn(stub, adminPwd, token, amount)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success([]byte("success"))
}

//对账
//func (a *amicoPayChaincode) check(stub shim.ChaincodeStubInterface, args []string) pb.Response {
//
//}


//工具类
func md5encode(str string) string {
	h := md5.New()
	io.WriteString(h, str)
	return fmt.Sprintf("%x", h.Sum(nil))
}
//func md5encode(input string) pb.Response {
//	h := md5.New()
//	h.Write([]byte(input)) // 加密
//	cipherStr := h.Sum(nil)
//	return hex.EncodeToString(cipherStr) // 输出加密结果
//}

func isUserExist(stub shim.ChaincodeStubInterface,userId string) (existed bool, e error) {
	userKey, err := stub.CreateCompositeKey("User", []string{"user", userId})
	if err != nil {
		return false,errors.New("userKet getting error")
	}
	userAsBytes, err := stub.GetState(userKey)
	if err != nil {
		return false,errors.New("db error")
	}else if userAsBytes == nil {
		return false,nil
	}
	return true,nil
}

//转账交易
/*input:token string
        from-userid string
        to-userid string
        amount float32
        password string
        needApproval list
        remark string
  output: transactionId(if successed)
		  wrong-pwd
          unexist-from-userid
          unexist-to-userid
		  insufficient-balance
          improper-amount
		  imporper-approval
		  unexist-approval-userid
*/
func (a *AmicoPayChaincode) createTransaction(stub shim.ChaincodeStubInterface, token string, fromUserId string, toUserId string, amount float64,
	password string, needApproval []string, remark string) (transaction Transaction, e error) {

	//获取公共参数
	parasAsBytes, err := stub.GetState("Paras")
	if err != nil {
		return transaction, errors.New("db get error")
	}
	var paras Paras
	err = json.Unmarshal(parasAsBytes, &paras)
	if err != nil {
		return transaction, errors.New("unmarshall failed")
	}
	transactionRate := paras.TransactionRate

	//检查from用户是否存在
	fromUserKey, err := stub.CreateCompositeKey("User", []string{"user", fromUserId})
	if err != nil {
		return transaction, errors.New("fromUserKey composition error")
	}
	fromUserAsBytes, err := stub.GetState(fromUserKey)
	if err != nil {
		return transaction, errors.New("db get error")
	}else if fromUserAsBytes == nil {
		return transaction, errors.New("This user doesn't exists: " + fromUserId)
	}

	//检查to用户是否存在
	toUserKey, err := stub.CreateCompositeKey("User", []string{"user", toUserId})
	if err != nil {
		return transaction, errors.New("toUserKey composition error")
	}
	toUserAsBytes, err := stub.GetState(toUserKey)
	if err != nil {
		return transaction, errors.New("db get error")
	}else if toUserAsBytes == nil {
		return transaction, errors.New("This user doesn't exists: " + toUserId)
	}

	//检查余额是否足够
	var fromUser User
	err = json.Unmarshal(fromUserAsBytes, &fromUser)
	if err != nil{
		return transaction, errors.New("umarshall failed")
	}

	if fromUser.Balance[token] < amount {
		return transaction, errors.New("Insufficient account balance")
	}

	//检查密码是否正确
	signature := md5encode(password)
	if signature != fromUser.Signature {
		return transaction, errors.New("password incorrect")
	}

	//检查审核用户是否存在
	var userExist bool
	for _, approvalId := range needApproval {
		userExist,err = isUserExist(stub, approvalId)
		if !userExist  {
			return transaction, errors.New("approvalId doesn't existed")
		}
	}

	//资金冻结
	var toUser User
	err = json.Unmarshal(toUserAsBytes, &toUser)//fromUser数据已经在前面提取过
	if err != nil{
		return transaction, errors.New("umarshall failed")
	}
	transAmount := amount * (1-transactionRate)
	currentFromBalance := fromUser.Balance[token]
	currentFromPayableBalance := fromUser.PayableBalance[token]
	currentToRecivableBalance:= toUser.RecivableBalance[token]
	fromUser.Balance[token] = currentFromBalance - amount
	fromUser.PayableBalance[token] = currentFromPayableBalance + transAmount
	toUser.RecivableBalance[token] = currentToRecivableBalance + transAmount
	frozen := transAmount

	//已审核列表
	var Approved []string

	//交易id链表
	lastFromTransactionId := fromUser.LastTransactionIdList[token]
	lastToTransactionId := toUser.LastTransactionIdList[token]



	//构建transactionId
	transactionId,err := a.transactionIdGet(stub)
	if err != nil{
		return transaction, errors.New("transaction id get failed")
	}
	fromUser.LastTransactionIdList[token] = transactionId
	toUser.LastTransactionIdList[token] = transactionId

	//transaction构造
	transaction = Transaction{token,transactionId,
		fromUserId,toUserId,amount,frozen,needApproval,Approved,false,
		remark,lastFromTransactionId,lastToTransactionId}

	//结构体转json
	transactionJSONasBytes, err := json.Marshal(transaction)
	if err != nil {
		return transaction, errors.New("marshall failed")
	}
	fromUserJSONasBytes, err := json.Marshal(fromUser)
	if err != nil {
		return transaction, errors.New("marshall failed")
	}
	toUserJSONasBytes, err := json.Marshal(toUser)
	if err != nil {
		return transaction, errors.New("marshall failed")
	}

	//transaction提交
	err = stub.PutState(strconv.FormatInt(transactionId,10), transactionJSONasBytes)
	if err != nil {
		return transaction, errors.New("db put error")
	}
	err = stub.PutState(fromUserKey, fromUserJSONasBytes)
	if err != nil {
		return transaction, errors.New("db put error")
	}
	err = stub.PutState(toUserKey, toUserJSONasBytes)
	if err != nil {
		return transaction, errors.New("db put error")
	}

	//交易确认
	_, err =a.autoConfirm(stub, strconv.FormatInt(transactionId,10))
	if err != nil {
		return transaction, errors.New("confirm error")
	}

	return transaction,nil
}

//自动确认交易是否完成，完成则转账
func (a *AmicoPayChaincode) autoConfirm (stub shim.ChaincodeStubInterface, transactionId string) (finished bool, e error) {
	transactionAsBytes, err := stub.GetState(transactionId)
	if err != nil {
		return false,errors.New("db get error")
	}else if transactionAsBytes == nil {
		return false,errors.New("This transaction doesn't exists")
	}

	var transaction Transaction
	err = json.Unmarshal(transactionAsBytes, &transaction)
	if err != nil{
		return false,errors.New("umarshall failed")
	}

	needApprovalList := transaction.NeedApproval
	Approved := transaction.Approved

	allApproved := true

	for i := 0; i < len(needApprovalList); i++ {
		need := needApprovalList[i]
		if(need != "approved") {
			allApproved = false
			for j:=0; j<len(Approved); j++ {
				if(need == Approved[j]) {
					needApprovalList[i]="approved"
					continue
				}
			}
		}
	}

	finished = false //是否交易完成

	//审核若全部通过，执行转账
	if(allApproved) {
		fromUserId := transaction.FromUserId
		toUserId := transaction.ToUserId
		//检查from用户是否存在
		fromUserKey, err := stub.CreateCompositeKey("User", []string{"user", fromUserId})
		if err != nil {
			return false,errors.New("fromkUserKey composition error")
		}
		fromUserAsBytes, err := stub.GetState(fromUserKey)
		if err != nil {
			return false,errors.New("db get error")
		}else if fromUserAsBytes == nil {
			return false,errors.New("This user doesn't exists: " + fromUserId)
		}

		//检查to用户是否存在
		toUserKey, err := stub.CreateCompositeKey("User", []string{"user", toUserId})
		if err != nil {
			return false,errors.New("toUserKey composition error")
		}
		toUserAsBytes, err := stub.GetState(toUserKey)
		if err != nil {
			return false,errors.New("db get error")
		}else if toUserAsBytes == nil {
			return false,errors.New("This user doesn't exists: " + toUserId)
		}

		//解析
		var fromUser User
		var toUser User
		err =json.Unmarshal(fromUserAsBytes, &fromUser)
		if err != nil{
			return false,errors.New("umarshall failed")
		}
		err =json.Unmarshal(toUserAsBytes, &toUser)
		if err != nil{
			return false,errors.New("umarshall failed")
		}

		token := transaction.TokenType
		frozen := transaction.Frozen
		currentToBalance := toUser.Balance[token]
		currentFromPayableBalance := fromUser.PayableBalance[token]
		currentToRecivableBalance := toUser.RecivableBalance[token]

		fromUser.PayableBalance[token] = currentFromPayableBalance- frozen
		toUser.RecivableBalance[token] = currentToRecivableBalance - frozen
		toUser.Balance[token] = currentToBalance + frozen

		transaction.Frozen=0
		transaction.Completed = true

		//user结构体转json
		fromUserJSONasBytes, err := json.Marshal(fromUser)
		if err != nil {
			return false,errors.New("marshall failed")
		}
		toUserJSONasBytes, err := json.Marshal(toUser)
		if err != nil {
			return false,errors.New("marshall failed")
		}

		//上传
		err = stub.PutState(fromUserKey, fromUserJSONasBytes)
		if err != nil {
			return false,errors.New("db put error")
		}
		err = stub.PutState(toUserKey, toUserJSONasBytes)
		if err != nil {
			return false,errors.New("db put error")
		}

		finished = true
	}

	//存储上传
	//结构体转json
	transactionJSONasBytes, err := json.Marshal(transaction)
	if err != nil {
		return false,errors.New("marshall failed")
	}


	//transaction提交
	err = stub.PutState(transactionId, transactionJSONasBytes)
	if err != nil {
		return false,errors.New("marshall failed")
	}

	return finished,nil
}

// main函数在实例化时启动容器中的链码
func main() {
	err := shim.Start(new(AmicoPayChaincode))
	if err != nil {
		fmt.Printf("Error starting Simple chaincode: %s", err)
	}
}
