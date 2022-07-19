from iconservice import *

TAG = 'Governance'


class SystemInterface(InterfaceScore):
    @interface
    def setRevision(self, code: int):
        pass

    @interface
    def acceptScore(self, txHash: bytes):
        pass

    @interface
    def rejectScore(self, txHash: bytes):
        pass

    @interface
    def blockScore(self, address: Address):
        pass

    @interface
    def unblockScore(self, address: Address):
        pass

    @interface
    def setStepPrice(self, price: int):
        pass

    @interface
    def setStepCost(self, type: str, cost: int):
        pass

    @interface
    def setMaxStepLimit(self, contextType: str, limit: int):
        pass

    @interface
    def addDeployer(self, address: Address):
        pass

    @interface
    def removeDeployer(self, address: Address):
        pass

    @interface
    def grantValidator(self, address: Address):
        pass

    @interface
    def revokeValidator(self, address: Address):
        pass

    @interface
    def addMember(self, address: Address):
        pass

    @interface
    def removeMember(self, address: Address):
        pass

    @interface
    def addLicense(self, contentId: str):
        pass

    @interface
    def removeLicense(self, contentId: str):
        pass

    @interface
    def setTimestampThreshold(self, threshold: int):
        pass

    @interface
    def setRoundLimitFactor(self, factor: int):
        pass

    @interface
    def setDeployerWhiteListEnabled(self, yn: bool):
        pass

    @interface
    def setMinimizeBlockGen(self, yn: bool):
        pass

    @interface
    def setUseSystemDeposit(self, address: Address, yn: bool):
        pass

    @interface
    def openBTPNetwork(self, networkTypeName: str, name: str, owner: Address):
        pass

    @interface
    def closeBTPNetwork(self, id: int):
        pass


class Governance(IconScoreBase):

    def __init__(self, db: IconScoreDatabase) -> None:
        super().__init__(db)
        self.system_score = self.create_interface_score(ZERO_SCORE_ADDRESS, SystemInterface)

    def on_install(self, name: str, value: int) -> None:
        Logger.info(f'install governance: {name}, value: {value}')
        super().on_install()

    def on_update(self) -> None:
        Logger.info('update governance')
        super().on_update()

    # NOTE: Governance should not accept incoming funds
    # @payable
    # def fallback(self):
    #     """
    #     Called when anyone sends funds to the SCORE.
    #     """
    #     Logger.debug(f'FundTransfer({self.msg.sender}, {self.msg.value})', TAG)

    @external
    def setRevision(self, code: int):
        self.system_score.setRevision(code)

    @external
    def acceptScore(self, txHash: bytes):
        self.system_score.acceptScore(txHash)

    @external
    def rejectScore(self, txHash: bytes):
        self.system_score.rejectScore(txHash)

    @external
    def blockScore(self, address: Address):
        self.system_score.blockScore(address)

    @external
    def unblockScore(self, address: Address):
        self.system_score.unblockScore(address)

    @external
    def setStepPrice(self, price: int):
        self.system_score.setStepPrice(price)

    @external
    def setStepCost(self, type: str, cost: int):
        self.system_score.setStepCost(type, cost)

    @external
    def setMaxStepLimit(self, contextType: str, limit: int):
        self.system_score.setMaxStepLimit(contextType, limit)

    @external
    def grantValidator(self, address: Address):
        self.system_score.grantValidator(address)

    @external
    def revokeValidator(self, address: Address):
        self.system_score.revokeValidator(address)

    @external
    def addMember(self, address: Address):
        self.system_score.addMember(address)

    @external
    def removeMember(self, address: Address):
        self.system_score.removeMember(address)

    @external
    def addDeployer(self, address: Address):
        self.system_score.addDeployer(address)

    @external
    def removeDeployer(self, address: Address):
        self.system_score.removeDeployer(address)

    @external
    def addLicense(self, contentId: str):
        self.system_score.addLicense(contentId)

    @external
    def removeLicense(self, contentId: str):
        self.system_score.removeLicense(contentId)

    @external
    def setTimestampThreshold(self, threshold: int):
        self.system_score.setTimestampThreshold(threshold)

    @external
    def setRoundLimitFactor(self, factor: int):
        self.system_score.setRoundLimitFactor(factor)

    @external
    def setDeployerWhiteListEnabled(self, yn: bool):
        self.system_score.setDeployerWhiteListEnabled(yn)

    @external
    def setMinimizeBlockGen(self, yn: bool):
        self.system_score.setMinimizeBlockGen(yn)

    @external
    def setUseSystemDeposit(self, address: Address, yn: bool):
        self.system_score.setUseSystemDeposit(address, yn)

    @external(readonly=True)
    def updated(self) -> bool:
        return True

    @external
    def openBTPNetwork(self, networkTypeName: str, name: str, owner: Address):
        self.system_score.openBTPNetwork(networkTypeName, name, owner)

    @external
    def closeBTPNetwork(self, id: int):
        self.system_score.closeBTPNetwork(id)
