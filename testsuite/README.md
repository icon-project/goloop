# TEST Suite
## Environment
####`Mac`
OpenJDK "1.8.0_212"<br>
OpenJDK Runtime Environment (AdoptOpenJDK)(build 1.8.0_212-b03)<br>
OpenJDK 64-Bit Server VM (AdoptOpenJDK)(build 25.212-b03, mixed mode)

####`CI`
OpenJDK version “1.8.0_201”<br>
OpenJDK Runtime Environment (IcedTea 3.11.0) (Alpine 8.201.08-r1)<br>
OpenJDK 64-Bit Server VM (build 25.201-b08, mixed mode)

## 패키지 구성
|data directory||
|:---|:---|
|genesisStorage|genesis file & governance score|
|scores|test scores|
|etc.|configuration files & key|

|java directory||
|:---|:---|
|foundation.icon.test||
|cases|테스트 케이스|
|common|공통적으로 사용되는 클래스|
|scores|score wrapping class|

## 테스트 클래스
`BasicScoreTest, MultiSigWalletTest` icon에서 score테스트를 위해 사용중이던 기본 테스트<br>

##### `BtpApiTest` BTP관련 기능 테스트
| 테스트함수 |
|:---|
|verifyVotes|
|ApiTest|

##### `ChainScoreTest` chainscore기능 테스트
| 테스트함수 |
|:---|
|disableEnableScore|
|setRevision|
|acceptScore|
|rejectScore|
|blockUnblockScore|
|setStepPrice|
|setStepCost|
|setMaxStepLimit|
|grantRevokeValidator|
|addRemoveMember|
|addRemoveDeployer|

##### `DeployTest` deploy관련 기능 테스트
| 테스트함수 |
|:---|
|notEnoughBalance|
|notEnoughStepLimit|
|installWithInvalidParams|
|updateWithInvalidParams|
|installScoreAndCall|
|updateScoreAndCall|
|updateWithInvalidOwner|
|updateToInvalidScoreAddress|
|invalidContentNoRootFile|
|invalidContentNotZip|
|invalidContentTooBig|
|invalidScoreNoOnInstallMethod|
|invalidScoreNoOnUpdateMethod|
|invalidSignature|
|deployGovScore|

##### `GetAPITest` json rpc의 icx_getScoreAPI 기능 테스트
| 테스트함수 |
|:---|
|testGetAPIForStepCounter|
|validateGetScoreApi|
|notExistsScoreAddress|
|getApiWithEOA|

##### `GetTotalSupplyTest` json rpc의 icx_getTotalSupply 기능 테스트
| 테스트함수 |
|:---|
|testGetTotalSupply|

##### `ReceiptTest` receipt 기능 테스트
| 테스트함수 |
|:---|
|eventLog|
|interCallEventLog|
|logsBloomWithNoIndex|
|logsBloomWithIndex|
|transferTxResultParams|
|deployTxResultParams|
|callTxResultParams|
|transferTxByHashParams|
|callTxByHashParams|

##### `RevertTest` transaction실패 시 transaction실행 상태로 정상 복구 되는 지 확인
| 테스트함수 |
|:---|
|testRevert|

##### `ScoreParamTest` call transaction관련 기능 테스트
| 테스트함수 |
|:---|
|callInt|
|callStr|
|callBytes|
|callBool|
|callAddress|
|callAll|
|interCallBool|
|interCallAddress|
|interCallInt|
|interCallBytes|
|interCallStr|
|interCallAll|
|invalidInterCallBool|
|invalidInterCallAddress|
|invalidInterCallBytes|
|invalidInterCallStr|
|invalidInterCallInt|
|callDefaultParam|
|interCallDefaultParam|
|interCallWithNull|
|interCallWithMoreParams|
|invalidAddUndefinedParam|
|interCallWithEmptyString|
|interCallWithDefaultParam|

##### `ScoreTest` call transaction관련 기능 테스트
| 테스트함수 |
|:---|
|invalidScoreAddr|
|invalidParamName|
|unexpectedParam|
|notEnoughStepLimit|
|notEnoughBalance|
|callWithValue|
|timeoutCallInfiniteLoop|
|infiniteInterCall|
|invalidSignature|
|notEnoughBalToCall|

##### `ScoreTestNormal` call transaction관련 기능 테스트
| 테스트함수 |
|:---|
|invalidMethodName|
|invalidParamName|
|unexpectedParam|

##### `StepTest` step 변경 & 변경된 step에 맞게 동작하는지 테스트
| 테스트함수 |
|:---|
|transferStep|
|deployStep|
|varDb - edge case포함|

edge case : transaction실행 중 step이 모자라면 해당 operation을 실행하지 않고 step을 차감하고 종료 한다.<br>
ex) step이 1남은 상태에서 step이  2인 opreation을 실행하려 할때 1을 차감하고 해당 operation은 실행하지 않고 종료.

##### `TransferTest` coin전송관련 기능 테스트
| 테스트함수 |
|:---|
|notEnoughBalance|
|notEnoughStepLimit|
|invalidSignature|
|transferAndCheckBal|
|transferWithMessage|

##### `WSEventTest` BTP관련 기능 테스트
| 테스트함수 |
|:---|
|wsBlkMonitorTest|
|wsEvtMonitorTest|

##Running Option
Audit - Audit 기능 enable/disable<br>
default disable<br>
`ex) ./gradlew testGovernance -DAUDIT="true"`

## Tag
#### TAG_GOVERNANCE
`@Tag(Constants.TAG_GOVERNANCE)`<br>
governance call을 통해 block chain의 환경이 변경될 수 있는 테스트 (ex. stepCost, stepPrice, ...)<br>
주로 fee와 관련되어 맞게 차감되는지 혹은 fee와 관련된 negative테스트 들로 이루어 진다<br>
ChainScoreTest, DeployTest, ScoreTest, StepTest, TransferTest<br>
 
#### TAG_NORMAL
`@Tag(Constants.TAG_NORMAL)`<br>
governance call에 의해 block chain 환경이 변하지 않는 테스트<br>
BasicScoreTest, GetAPITest, GetTotalSupplyTest, MultiSigWalletTest, RevertTest, WSEventTest<br>